package analyzer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

type Analyzer struct {
	Catalog *model.Catalog
}

type Result struct {
	Query       parser.Query
	Columns     []ResultColumn
	Params      []ResultParam
	Diagnostics []Diagnostic
}

type ResultColumn struct {
	Name     string
	Table    string
	GoType   string
	Nullable bool
}

type ResultParam struct {
	Name          string
	Style         parser.ParamStyle
	GoType        string
	Nullable      bool
	IsVariadic    bool
	VariadicCount int
}

type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

type diagKey struct {
	path    string
	line    int
	column  int
	message string
}

type scopeLookupResult int

const (
	scopeLookupOK scopeLookupResult = iota
	scopeLookupAliasNotFound
	scopeLookupColumnNotFound
	scopeLookupAmbiguous
)

type queryScope struct {
	entries map[string]*scopeEntry
}

type scopeEntry struct {
	name    string
	columns map[string]scopeColumn
}

type scopeColumn struct {
	name     string
	owner    string
	goType   string
	nullable bool
}

type textIndex struct {
	sql         string
	lineOffsets []int
}

type paramInfo struct {
	GoType   string
	Nullable bool
}

type posKey struct {
	line   int
	column int
}

func New(catalog *model.Catalog) *Analyzer {
	return &Analyzer{Catalog: catalog}
}

func (a *Analyzer) Analyze(q parser.Query) Result {
	result := Result{
		Query:   q,
		Columns: make([]ResultColumn, 0, len(q.Columns)),
		Params:  make([]ResultParam, 0, len(q.Params)),
	}

	seen := make(map[diagKey]struct{})
	addDiag := func(d Diagnostic) {
		key := diagKey{path: d.Path, line: d.Line, column: d.Column, message: d.Message}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result.Diagnostics = append(result.Diagnostics, d)
	}

	for _, pd := range q.Diagnostics {
		addDiag(Diagnostic{
			Path:     pd.Path,
			Line:     pd.Line,
			Column:   pd.Column,
			Message:  pd.Message,
			Severity: convertSeverity(pd.Severity),
		})
	}

	catalog := a.Catalog
	hasCatalog := catalog != nil

	baseScope := newQueryScope()
	if hasCatalog {
		for name, table := range catalog.Tables {
			entry := scopeEntryFromTable(table)
			baseScope.addEntry(name, entry)
		}
	} else {
		addDiag(Diagnostic{
			Path:     q.Block.Path,
			Line:     q.Block.Line,
			Column:   q.Block.Column,
			Message:  "schema catalog unavailable; using default column and parameter types",
			Severity: SeverityWarning,
		})
	}

	for _, cte := range q.CTEs {
		entry, diags := a.resolveCTE(cte, q, baseScope, hasCatalog)
		for _, d := range diags {
			addDiag(d)
		}
		if entry != nil {
			baseScope.addEntry(cte.Name, entry)
		}
	}

	tokens, err := tokenizer.Scan(q.Block.Path, []byte(q.Block.SQL), false)
	if err != nil {
		tokens = nil
	}

	workingScope := baseScope.clone()
	if tokens != nil {
		addAliasesFromTokens(workingScope, tokens)
	}

	for _, col := range q.Columns {
		rc, diags := resolveResultColumn(col, workingScope, q.Block, hasCatalog)
		result.Columns = append(result.Columns, rc)
		for _, d := range diags {
			addDiag(d)
		}
	}

	paramInfos := inferParamTypes(a.Catalog, q, workingScope)
	for idx, param := range q.Params {
		rp := ResultParam{
			Name:          param.Name,
			Style:         param.Style,
			GoType:        "interface{}",
			Nullable:      true,
			IsVariadic:    param.IsVariadic,
			VariadicCount: param.VariadicCount,
		}
		if info, ok := paramInfos[idx]; ok {
			rp.GoType = info.GoType
			rp.Nullable = info.Nullable
		}
		result.Params = append(result.Params, rp)
	}

	return result
}

func convertSeverity(s parser.Severity) Severity {
	switch s {
	case parser.SeverityError:
		return SeverityError
	case parser.SeverityWarning:
		return SeverityWarning
	default:
		return SeverityWarning
	}
}

func columnDisplayName(col parser.Column) string {
	if col.Alias != "" {
		return col.Alias
	}
	if col.Expr != "" {
		return col.Expr
	}
	return ""
}

func deriveColumnName(col parser.Column) string {
	expr := strings.TrimSpace(col.Expr)
	if expr == "" {
		return ""
	}
	if idx := strings.LastIndex(expr, "."); idx >= 0 {
		expr = expr[idx+1:]
	}
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	return tokenizer.NormalizeIdentifier(expr)
}

func lookupTable(cat *model.Catalog, name string) *model.Table {
	if cat == nil || name == "" {
		return nil
	}
	if tbl, ok := cat.Tables[name]; ok {
		return tbl
	}
	for key, tbl := range cat.Tables {
		if strings.EqualFold(key, name) {
			return tbl
		}
	}
	return nil
}

func lookupColumn(tbl *model.Table, name string) *model.Column {
	if tbl == nil || name == "" {
		return nil
	}
	for _, col := range tbl.Columns {
		if strings.EqualFold(col.Name, name) {
			return col
		}
	}
	return nil
}

func (a *Analyzer) resolveCTE(cte parser.CTE, parent parser.Query, scope *queryScope, hasCatalog bool) (*scopeEntry, []Diagnostic) {
	diags := make([]Diagnostic, 0, 2)

	anchorSQL, recursiveSQL, hasUnion := splitRecursiveParts(cte.SelectSQL)
	if strings.TrimSpace(anchorSQL) == "" {
		diags = append(diags, Diagnostic{
			Path:     parent.Block.Path,
			Line:     cte.Line,
			Column:   cte.Column,
			Message:  fmt.Sprintf("CTE %s is missing a SELECT body", cte.Name),
			Severity: SeverityError,
		})
		return nil, diags
	}

	blk := block.Block{
		Path:   parent.Block.Path,
		Line:   cte.Line,
		Column: cte.Column,
		SQL:    anchorSQL,
	}

	anchorQuery, anchorDiags := parser.Parse(blk)
	for _, pd := range anchorDiags {
		severity := convertSeverity(pd.Severity)
		if severity == SeverityError && strings.Contains(pd.Message, "requires alias") {
			severity = SeverityWarning
		}
		diags = append(diags, Diagnostic{
			Path:     pd.Path,
			Line:     pd.Line,
			Column:   pd.Column,
			Message:  pd.Message,
			Severity: severity,
		})
	}

	if anchorQuery.Verb != parser.VerbSelect {
		diags = append(diags, Diagnostic{
			Path:     parent.Block.Path,
			Line:     cte.Line,
			Column:   cte.Column,
			Message:  fmt.Sprintf("CTE %s must be defined with a SELECT statement", cte.Name),
			Severity: SeverityError,
		})
		return nil, diags
	}

	workingScope := scope.clone()
	anchorTokens, err := tokenizer.Scan(parent.Block.Path, []byte(anchorSQL), false)
	if err == nil {
		addAliasesFromTokens(workingScope, anchorTokens)
	}

	columnNames, colDiags := cteOutputNames(cte, anchorQuery)
	diags = append(diags, colDiags...)
	if len(columnNames) == 0 {
		return nil, diags
	}

	resolved := make([]scopeColumn, 0, len(columnNames))
	for idx, col := range anchorQuery.Columns {
		name := columnNames[idx]
		sc := scopeColumn{name: name, owner: cte.Name, goType: "interface{}", nullable: true}

		suppressDefaultWarning := false
		if hasCatalog {
			alias := col.Table
			columnName := deriveColumnName(col)
			agg, isAggregate := parseAggregateExpr(col.Expr)
			skipLookup := false
			if isAggregate {
				if agg.argStar {
					sc.goType = "int64"
					sc.nullable = false
					skipLookup = true
				} else if agg.argColumn == "" {
					diags = append(diags, Diagnostic{
						Path:     parent.Block.Path,
						Line:     col.Line,
						Column:   col.Column,
						Message:  fmt.Sprintf("unable to infer metadata for aggregate %s in CTE %s; defaulting to interface{}", col.Expr, cte.Name),
						Severity: SeverityWarning,
					})
					suppressDefaultWarning = true
					skipLookup = true
				} else {
					alias = agg.argAlias
					columnName = agg.argColumn
				}
			}

			if !skipLookup {
				lookup, _, res := workingScope.lookup(alias, columnName)
				switch res {
				case scopeLookupOK:
					if isAggregate {
						goType, nullable, ok := aggregateResultFromOperand(agg.kind, lookup)
						if !ok {
							diags = append(diags, Diagnostic{
								Path:     parent.Block.Path,
								Line:     col.Line,
								Column:   col.Column,
								Message:  fmt.Sprintf("unable to infer Go type for aggregate %s in CTE %s; defaulting to interface{}", col.Expr, cte.Name),
								Severity: SeverityWarning,
							})
							sc.goType = "interface{}"
							sc.nullable = true
							suppressDefaultWarning = true
						} else {
							sc.goType = goType
							sc.nullable = nullable
						}
					} else {
						sc.goType = lookup.goType
						sc.nullable = lookup.nullable
					}
				case scopeLookupAliasNotFound:
					if alias != "" {
						diags = append(diags, Diagnostic{
							Path:     parent.Block.Path,
							Line:     col.Line,
							Column:   col.Column,
							Message:  fmt.Sprintf("CTE %s references unknown relation %s", cte.Name, alias),
							Severity: SeverityError,
						})
					} else if col.Alias != "" {
						diags = append(diags, Diagnostic{
							Path:     parent.Block.Path,
							Line:     col.Line,
							Column:   col.Column,
							Message:  fmt.Sprintf("CTE %s column %s derives from expression without schema mapping", cte.Name, col.Alias),
							Severity: SeverityWarning,
						})
					}
				case scopeLookupColumnNotFound:
					if alias != "" {
						diags = append(diags, Diagnostic{
							Path:     parent.Block.Path,
							Line:     col.Line,
							Column:   col.Column,
							Message:  fmt.Sprintf("CTE %s references unknown column %s.%s", cte.Name, alias, columnName),
							Severity: SeverityError,
						})
					} else if col.Alias != "" {
						diags = append(diags, Diagnostic{
							Path:     parent.Block.Path,
							Line:     col.Line,
							Column:   col.Column,
							Message:  fmt.Sprintf("CTE %s column %s derives from expression without schema mapping", cte.Name, col.Alias),
							Severity: SeverityWarning,
						})
					}
				case scopeLookupAmbiguous:
					diags = append(diags, Diagnostic{
						Path:     parent.Block.Path,
						Line:     col.Line,
						Column:   col.Column,
						Message:  fmt.Sprintf("CTE %s column %s is ambiguous; qualify with a table alias", cte.Name, name),
						Severity: SeverityError,
					})
				}
			}
		}

		if sc.goType == "interface{}" && hasCatalog && !suppressDefaultWarning {
			diags = append(diags, Diagnostic{
				Path:     parent.Block.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("unable to infer type for CTE column %s; defaulting to interface{}", name),
				Severity: SeverityWarning,
			})
		}

		resolved = append(resolved, sc)
	}

	if hasUnion && recursiveSQL != "" {

		recBlock := block.Block{
			Path:   parent.Block.Path,
			Line:   cte.Line,
			Column: cte.Column,
			SQL:    recursiveSQL,
		}
		recQuery, recDiags := parser.Parse(recBlock)
		for _, pd := range recDiags {
			severity := convertSeverity(pd.Severity)
			if severity == SeverityError && strings.Contains(pd.Message, "requires alias") {
				severity = SeverityWarning
			}
			diags = append(diags, Diagnostic{
				Path:     pd.Path,
				Line:     pd.Line,
				Column:   pd.Column,
				Message:  pd.Message,
				Severity: severity,
			})
		}
		projected := len(recQuery.Columns)
		if count, ok := selectColumnCount(recursiveSQL); ok {
			projected = count
		}
		if projected != len(resolved) {

			diags = append(diags, Diagnostic{
				Path:     parent.Block.Path,
				Line:     cte.Line,
				Column:   cte.Column,
				Message:  fmt.Sprintf("recursive term of CTE %s projects %d columns; expected %d", cte.Name, projected, len(resolved)),
				Severity: SeverityError,
			})
		}

	}

	entry := &scopeEntry{
		name:    cte.Name,
		columns: make(map[string]scopeColumn, len(resolved)),
	}
	for _, col := range resolved {
		entry.columns[normalizeIdent(col.name)] = col
	}

	return entry, diags
}

func cteOutputNames(cte parser.CTE, q parser.Query) ([]string, []Diagnostic) {
	if len(q.Columns) == 0 {
		return nil, []Diagnostic{(Diagnostic{
			Path:     q.Block.Path,
			Line:     q.Block.Line,
			Column:   q.Block.Column,
			Message:  fmt.Sprintf("CTE %s produced no columns", cte.Name),
			Severity: SeverityError,
		})}
	}

	if len(cte.Columns) > 0 {
		if len(cte.Columns) != len(q.Columns) {
			return nil, []Diagnostic{(Diagnostic{
				Path:     q.Block.Path,
				Line:     q.Block.Line,
				Column:   q.Block.Column,
				Message:  fmt.Sprintf("CTE %s lists %d columns but SELECT returned %d", cte.Name, len(cte.Columns), len(q.Columns)),
				Severity: SeverityError,
			})}
		}
		names := make([]string, len(cte.Columns))
		copy(names, cte.Columns)
		return names, nil
	}

	names := make([]string, len(q.Columns))
	diags := make([]Diagnostic, 0, len(q.Columns))
	for i, col := range q.Columns {
		name := columnDisplayName(col)
		if name == "" {
			name = deriveColumnName(col)
		}
		if name == "" {
			diags = append(diags, Diagnostic{
				Path:     q.Block.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("unable to determine name for column %d in CTE %s", i+1, cte.Name),
				Severity: SeverityError,
			})
			continue
		}
		names[i] = name
	}
	return names, diags
}

type aggregateKind int

type aggregateExpr struct {
	kind      aggregateKind
	argStar   bool
	argAlias  string
	argColumn string
}

const (
	aggregateKindUnknown aggregateKind = iota
	aggregateKindCount
	aggregateKindSum
	aggregateKindMin
	aggregateKindMax
	aggregateKindAvg
)

func parseAggregateExpr(expr string) (aggregateExpr, bool) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return aggregateExpr{}, false
	}
	open := strings.Index(trimmed, "(")
	if open < 0 {
		return aggregateExpr{}, false
	}
	funcName := strings.TrimSpace(trimmed[:open])
	if funcName == "" {
		return aggregateExpr{}, false
	}
	kind := aggregateKindUnknown
	switch strings.ToUpper(funcName) {
	case "COUNT":
		kind = aggregateKindCount
	case "SUM":
		kind = aggregateKindSum
	case "MIN":
		kind = aggregateKindMin
	case "MAX":
		kind = aggregateKindMax
	case "AVG":
		kind = aggregateKindAvg
	default:
		return aggregateExpr{}, false
	}

	close := strings.LastIndex(trimmed, ")")
	if close < 0 || close <= open {
		return aggregateExpr{}, false
	}
	after := strings.TrimSpace(trimmed[close+1:])
	if after != "" {
		return aggregateExpr{}, false
	}

	inner := strings.TrimSpace(trimmed[open+1 : close])
	if inner == "" {
		return aggregateExpr{kind: kind}, true
	}
	upperInner := strings.ToUpper(inner)
	if strings.HasPrefix(upperInner, "DISTINCT ") {
		inner = strings.TrimSpace(inner[len("DISTINCT "):])
	}

	agg := aggregateExpr{kind: kind}
	if inner == "*" {
		agg.argStar = true
		return agg, true
	}

	alias, column, ok := splitQualifiedIdentifier(inner)
	if ok {
		agg.argAlias = alias
		agg.argColumn = column
	}
	return agg, true
}

func splitQualifiedIdentifier(expr string) (string, string, bool) {
	tokens, err := tokenizer.Scan("", []byte(expr), false)
	if err != nil {
		return "", "", false
	}
	filtered := make([]tokenizer.Token, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Kind == tokenizer.KindEOF || tok.Kind == tokenizer.KindDocComment {
			continue
		}
		filtered = append(filtered, tok)
	}
	if len(filtered) == 1 && filtered[0].Kind == tokenizer.KindIdentifier {
		return "", tokenizer.NormalizeIdentifier(filtered[0].Text), true
	}
	if len(filtered) == 3 &&
		filtered[0].Kind == tokenizer.KindIdentifier &&
		filtered[1].Kind == tokenizer.KindSymbol && filtered[1].Text == "." &&
		filtered[2].Kind == tokenizer.KindIdentifier {
		alias := tokenizer.NormalizeIdentifier(filtered[0].Text)
		column := tokenizer.NormalizeIdentifier(filtered[2].Text)
		if column == "" {
			return "", "", false
		}
		return alias, column, true
	}
	return "", "", false
}

func aggregateKindString(kind aggregateKind) string {
	switch kind {
	case aggregateKindCount:
		return "COUNT"
	case aggregateKindSum:
		return "SUM"
	case aggregateKindMin:
		return "MIN"
	case aggregateKindMax:
		return "MAX"
	case aggregateKindAvg:
		return "AVG"
	default:
		return "aggregate"
	}
}

func aggregateResultFromOperand(kind aggregateKind, operand scopeColumn) (string, bool, bool) {
	switch kind {
	case aggregateKindCount:
		return "int64", false, true
	case aggregateKindSum:
		if isIntegerGoType(operand.goType) || isFloatGoType(operand.goType) {
			return operand.goType, true, true
		}
		return "", true, false
	case aggregateKindAvg:
		if isIntegerGoType(operand.goType) || isFloatGoType(operand.goType) {
			return "float64", true, true
		}
		return "", true, false
	case aggregateKindMin, aggregateKindMax:
		if operand.goType == "" || operand.goType == "interface{}" {
			return "", operand.nullable, false
		}
		return operand.goType, operand.nullable, true
	default:
		return "", operand.nullable, false
	}
}

func isIntegerGoType(goType string) bool {
	switch goType {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	default:
		return false
	}
}

func isFloatGoType(goType string) bool {
	switch goType {
	case "float32", "float64":
		return true
	default:
		return false
	}
}

func resolveResultColumn(col parser.Column, scope *queryScope, blk block.Block, hasCatalog bool) (ResultColumn, []Diagnostic) {
	rc := ResultColumn{
		Name:     columnDisplayName(col),
		Table:    col.Table,
		GoType:   "interface{}",
		Nullable: true,
	}
	diags := make([]Diagnostic, 0, 1)

	if !hasCatalog || scope == nil {
		return rc, diags
	}

	alias := col.Table
	columnName := deriveColumnName(col)

	agg, isAggregate := parseAggregateExpr(col.Expr)
	if isAggregate {
		if col.Alias == "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("aggregate %s requires an alias", aggregateKindString(agg.kind)),
				Severity: SeverityError,
			})
		}
		if agg.argStar {
			rc.GoType = "int64"
			rc.Nullable = false
			return rc, diags
		}
		if agg.argColumn == "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("unable to infer metadata for aggregate %s; defaulting to interface{}", col.Expr),
				Severity: SeverityWarning,
			})
			return rc, diags
		}
		alias = agg.argAlias
		columnName = agg.argColumn
	}

	lookup, entry, res := scope.lookup(alias, columnName)
	switch res {
	case scopeLookupOK:
		if rc.Table == "" {
			rc.Table = entry.name
		}
		if isAggregate {
			goType, nullable, ok := aggregateResultFromOperand(agg.kind, lookup)
			if !ok {
				diags = append(diags, Diagnostic{
					Path:     blk.Path,
					Line:     col.Line,
					Column:   col.Column,
					Message:  fmt.Sprintf("unable to infer Go type for aggregate %s; defaulting to interface{}", col.Expr),
					Severity: SeverityWarning,
				})
				rc.GoType = "interface{}"
				rc.Nullable = true
			} else {
				rc.GoType = goType
				rc.Nullable = nullable
			}
			return rc, diags
		}
		rc.GoType = lookup.goType
		rc.Nullable = lookup.nullable
	case scopeLookupAliasNotFound:
		if isAggregate {
			msg := fmt.Sprintf("aggregate %s references unknown relation", aggregateKindString(agg.kind))
			if alias != "" {
				msg = fmt.Sprintf("aggregate %s references unknown relation %s", aggregateKindString(agg.kind), alias)
			}
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  msg,
				Severity: SeverityError,
			})
			return rc, diags
		}
		if alias != "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q references unknown table %q", rcOrExprName(rc, col), alias),
				Severity: SeverityError,
			})
		} else if col.Alias != "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q derives from expression without schema mapping", col.Alias),
				Severity: SeverityWarning,
			})
		}
	case scopeLookupColumnNotFound:
		if isAggregate {
			target := columnName
			if target == "" {
				target = rcOrExprName(rc, col)
			}
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("aggregate %s could not resolve column %s", aggregateKindString(agg.kind), target),
				Severity: SeverityError,
			})
			return rc, diags
		}
		if alias != "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q references unknown column %s.%s", rcOrExprName(rc, col), alias, columnName),
				Severity: SeverityError,
			})
		} else if col.Alias != "" {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q derives from expression without schema mapping", col.Alias),
				Severity: SeverityWarning,
			})
		} else {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q could not be resolved", rcOrExprName(rc, col)),
				Severity: SeverityError,
			})
		}
	case scopeLookupAmbiguous:
		if isAggregate {
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("aggregate %s argument %s is ambiguous; qualify the column", aggregateKindString(agg.kind), rcOrExprName(rc, col)),
				Severity: SeverityError,
			})
			return rc, diags
		}
		diags = append(diags, Diagnostic{
			Path:     blk.Path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("result column %q is ambiguous; add a table alias", rcOrExprName(rc, col)),
			Severity: SeverityError,
		})
	}

	return rc, diags
}

func rcOrExprName(rc ResultColumn, col parser.Column) string {
	if rc.Name != "" {
		return rc.Name
	}
	if col.Expr != "" {
		return col.Expr
	}
	return "column"
}

func newQueryScope() *queryScope {
	return &queryScope{entries: make(map[string]*scopeEntry)}
}

func (s *queryScope) clone() *queryScope {
	if s == nil {
		return newQueryScope()
	}
	clone := newQueryScope()
	for key, entry := range s.entries {
		clone.entries[key] = entry
	}
	return clone
}

func (s *queryScope) addEntry(name string, entry *scopeEntry) {
	if s == nil || entry == nil {
		return
	}
	key := normalizeIdent(name)
	if key == "" {
		return
	}
	s.entries[key] = entry
}

func (s *queryScope) addAlias(alias string, entry *scopeEntry) {
	if s == nil || entry == nil {
		return
	}
	key := normalizeIdent(alias)
	if key == "" {
		return
	}
	s.entries[key] = entry
}

func (s *queryScope) get(name string) (*scopeEntry, bool) {
	if s == nil {
		return nil, false
	}
	entry, ok := s.entries[normalizeIdent(name)]
	return entry, ok
}

func (s *queryScope) lookup(alias, column string) (scopeColumn, *scopeEntry, scopeLookupResult) {
	if s == nil {
		return scopeColumn{}, nil, scopeLookupAliasNotFound
	}
	if alias != "" {
		entry, ok := s.entries[normalizeIdent(alias)]
		if !ok {
			return scopeColumn{}, nil, scopeLookupAliasNotFound
		}
		col, found := entry.columns[normalizeIdent(column)]
		if !found {
			return scopeColumn{}, entry, scopeLookupColumnNotFound
		}
		return col, entry, scopeLookupOK
	}

	if column == "" {
		return scopeColumn{}, nil, scopeLookupColumnNotFound
	}

	var (
		foundCol   scopeColumn
		foundEntry *scopeEntry
		matches    int
	)
	seen := make(map[*scopeEntry]struct{})
	for _, entry := range s.entries {
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		if col, ok := entry.columns[normalizeIdent(column)]; ok {
			foundCol = col
			foundEntry = entry
			matches++
			if matches > 1 {
				return scopeColumn{}, nil, scopeLookupAmbiguous
			}
		}
	}
	if matches == 1 {
		return foundCol, foundEntry, scopeLookupOK
	}
	return scopeColumn{}, nil, scopeLookupColumnNotFound
}

func scopeEntryFromTable(tbl *model.Table) *scopeEntry {
	cols := make(map[string]scopeColumn, len(tbl.Columns))
	for _, col := range tbl.Columns {
		cols[normalizeIdent(col.Name)] = scopeColumn{
			name:     col.Name,
			owner:    tbl.Name,
			goType:   SQLiteTypeToGo(col.Type),
			nullable: !col.NotNull,
		}
	}
	return &scopeEntry{name: tbl.Name, columns: cols}
}

func normalizeIdent(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(tokenizer.NormalizeIdentifier(name))
}

func addAliasesFromTokens(scope *queryScope, tokens []tokenizer.Token) {
	if scope == nil || len(tokens) == 0 {
		return
	}
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindEOF {
			break
		}
		if tok.Kind != tokenizer.KindKeyword {
			continue
		}
		text := strings.ToUpper(tok.Text)
		if text != "FROM" && text != "JOIN" {
			continue
		}
		i++
		relation, ok := parseRelationName(tokens, &i)
		if !ok {
			continue
		}
		entry, exists := scope.get(relation)
		if !exists {
			continue
		}
		alias := parseAlias(tokens, &i)
		if alias != "" {
			scope.addAlias(alias, entry)
		}
	}
}

func parseRelationName(tokens []tokenizer.Token, idx *int) (string, bool) {
	i := *idx
	for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
		i++
	}
	if i >= len(tokens) {
		*idx = i
		return "", false
	}
	tok := tokens[i]
	if tok.Kind != tokenizer.KindIdentifier {
		*idx = i
		return "", false
	}
	name := tokenizer.NormalizeIdentifier(tok.Text)
	i++
	*idx = i
	return name, true
}

func parseAlias(tokens []tokenizer.Token, idx *int) string {
	i := *idx
	for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
		i++
	}
	if i >= len(tokens) {
		*idx = i
		return ""
	}
	if tokens[i].Kind == tokenizer.KindKeyword && strings.ToUpper(tokens[i].Text) == "AS" {
		i++
		for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
			i++
		}
	}
	if i < len(tokens) && tokens[i].Kind == tokenizer.KindIdentifier {
		alias := tokenizer.NormalizeIdentifier(tokens[i].Text)
		i++
		*idx = i
		return alias
	}
	*idx = i
	return ""
}

func selectColumnCount(sql string) (int, bool) {
	tokens, err := tokenizer.Scan("", []byte(sql), false)
	if err != nil {
		return 0, false
	}
	encounteredSelect := false
	depth := 0
	count := 0
	sawExpr := false
	for _, tok := range tokens {
		if tok.Kind == tokenizer.KindEOF {
			break
		}
		if !encounteredSelect {
			if tok.Kind == tokenizer.KindKeyword && strings.ToUpper(tok.Text) == "SELECT" {
				encounteredSelect = true
			}
			continue
		}
		if tok.Kind == tokenizer.KindKeyword && depth == 0 {
			upper := strings.ToUpper(tok.Text)
			if upper == "FROM" {
				if sawExpr {
					count++
				}
				return count, true
			}
			if (upper == "DISTINCT" || upper == "ALL") && !sawExpr {
				continue
			}
		}
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			case ",":
				if depth == 0 {
					count++
					sawExpr = false
					continue
				}
			}
		}
		if tok.Kind != tokenizer.KindDocComment {
			sawExpr = true
		}
	}
	if !encounteredSelect {
		return 0, false
	}
	if sawExpr {
		count++
	}
	return count, true
}

func splitRecursiveParts(sql string) (string, string, bool) {

	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return "", "", false
	}
	tokens, err := tokenizer.Scan("", []byte(trimmed), false)
	if err != nil {
		return trimmed, "", false
	}
	idx := newTextIndex(trimmed)
	depth := 0
	unionIdx := -1
	for i, tok := range tokens {
		if tok.Kind == tokenizer.KindEOF {
			break
		}

		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			}
			continue
		}
		if depth == 0 {
			upper := strings.ToUpper(tok.Text)
			if (tok.Kind == tokenizer.KindKeyword || tok.Kind == tokenizer.KindIdentifier) && upper == "UNION" {
				unionIdx = i
				break
			}
		}
	}

	if unionIdx == -1 {
		return trimmed, "", false
	}
	unionOffset := idx.offset(tokens[unionIdx])
	anchor := strings.TrimSpace(trimmed[:unionOffset])

	after := unionIdx + 1
	for after < len(tokens) {
		if tokens[after].Kind == tokenizer.KindDocComment {
			after++
			continue
		}
		if tokens[after].Kind == tokenizer.KindKeyword {
			upper := strings.ToUpper(tokens[after].Text)
			if upper == "ALL" || upper == "DISTINCT" {
				after++
				continue
			}
		}
		break
	}
	if after >= len(tokens) || tokens[after].Kind == tokenizer.KindEOF {
		return anchor, "", true
	}
	recursiveStart := idx.offset(tokens[after])
	if recursiveStart >= len(trimmed) {
		return anchor, "", true
	}
	recursive := strings.TrimSpace(trimmed[recursiveStart:])
	return anchor, recursive, true
}

func newTextIndex(sql string) textIndex {
	offsets := make([]int, 1, strings.Count(sql, "\n")+2)
	offsets[0] = 0
	for i := 0; i < len(sql); {
		switch sql[i] {
		case '\r':
			i++
			if i < len(sql) && sql[i] == '\n' {
				i++
			}
			offsets = append(offsets, i)
		case '\n':
			i++
			offsets = append(offsets, i)
		default:
			_, size := utf8.DecodeRuneInString(sql[i:])
			i += size
		}
	}
	return textIndex{sql: sql, lineOffsets: offsets}
}

func (p textIndex) offset(tok tokenizer.Token) int {
	line := tok.Line
	if line <= 0 {
		return 0
	}
	if line-1 >= len(p.lineOffsets) {
		return len(p.sql)
	}
	base := p.lineOffsets[line-1]
	col := tok.Column
	idx := base
	for count := 1; count < col && idx < len(p.sql); count++ {
		_, size := utf8.DecodeRuneInString(p.sql[idx:])
		idx += size
	}
	if idx > len(p.sql) {
		return len(p.sql)
	}
	return idx
}

func inferParamTypes(cat *model.Catalog, q parser.Query, scope *queryScope) map[int]paramInfo {
	if cat == nil || len(q.Params) == 0 {
		return nil
	}

	tokens, err := tokenizer.Scan(q.Block.Path, []byte(q.Block.SQL), false)
	if err != nil {
		return nil
	}

	positions := make(map[posKey]int, len(tokens))
	for idx, tok := range tokens {
		positions[posKey{line: actualTokenLine(q, tok), column: tok.Column}] = idx
	}

	tokenIndexByParam := make(map[int]int, len(q.Params))
	paramIndexByToken := make(map[int]int, len(q.Params))
	for idx, param := range q.Params {
		if tokenIdx, ok := positions[posKey{line: param.Line, column: param.Column}]; ok {
			tokenIndexByParam[idx] = tokenIdx
			paramIndexByToken[tokenIdx] = idx
		}
	}
	if len(tokenIndexByParam) == 0 {
		return nil
	}

	infos := make(map[int]paramInfo, len(tokenIndexByParam))
	for paramIdx, tokenIdx := range tokenIndexByParam {
		if _, exists := infos[paramIdx]; exists {
			continue
		}
		table, column, ok := matchEqualityReference(tokens, tokenIdx)
		if !ok {
			continue
		}
		if scope != nil {
			if resolved, _, status := scope.lookup(table, column); status == scopeLookupOK && resolved.goType != "interface{}" {
				infos[paramIdx] = paramInfo{GoType: resolved.goType, Nullable: resolved.nullable}
				continue
			} else if status == scopeLookupAliasNotFound && column != "" {
				if fallback, _, fbStatus := scope.lookup("", column); fbStatus == scopeLookupOK && fallback.goType != "interface{}" {
					infos[paramIdx] = paramInfo{GoType: fallback.goType, Nullable: fallback.nullable}
					continue
				}
			}
		}

		if info, found := schemaInfoForColumn(cat, table, column); found {
			infos[paramIdx] = info
		}
	}

	if q.Verb == parser.VerbInsert {
		inferInsertParams(cat, tokens, paramIndexByToken, infos)
	}

	if len(infos) == 0 {
		return nil
	}
	return infos
}

func schemaInfoForColumn(cat *model.Catalog, tableName, columnName string) (paramInfo, bool) {
	table := lookupTable(cat, tableName)
	if table == nil {
		return paramInfo{}, false
	}
	column := lookupColumn(table, columnName)
	if column == nil {
		return paramInfo{}, false
	}
	return paramInfo{
		GoType:   SQLiteTypeToGo(column.Type),
		Nullable: !column.NotNull,
	}, true
}

func matchEqualityReference(tokens []tokenizer.Token, paramIdx int) (string, string, bool) {
	if table, column, ok := equalityLeftReference(tokens, paramIdx); ok {
		return table, column, true
	}
	if table, column, ok := equalityRightReference(tokens, paramIdx); ok {
		return table, column, true
	}
	return "", "", false
}

func equalityLeftReference(tokens []tokenizer.Token, paramIdx int) (string, string, bool) {
	if paramIdx == 0 {
		return "", "", false
	}
	eqIdx := paramIdx - 1
	if eqIdx < 0 || eqIdx >= len(tokens) {
		return "", "", false
	}
	eqTok := tokens[eqIdx]
	if eqTok.Kind != tokenizer.KindSymbol || eqTok.Text != "=" {
		return "", "", false
	}
	return parseColumnReferenceBackward(tokens, eqIdx-1)
}

func equalityRightReference(tokens []tokenizer.Token, paramIdx int) (string, string, bool) {
	nextIdx := paramIdx + 1
	if nextIdx >= len(tokens) {
		return "", "", false
	}

	tok := tokens[paramIdx]
	if tok.Kind == tokenizer.KindSymbol {
		switch tok.Text {
		case ":":
			if nextIdx < len(tokens) && tokens[nextIdx].Kind == tokenizer.KindIdentifier {
				nextIdx++
			}
		case "?":
			if nextIdx < len(tokens) && tokens[nextIdx].Kind == tokenizer.KindNumber {
				nextIdx++
			}
		}
	}

	if nextIdx >= len(tokens) {
		return "", "", false
	}
	eqTok := tokens[nextIdx]
	if eqTok.Kind != tokenizer.KindSymbol || eqTok.Text != "=" {
		return "", "", false
	}
	return parseColumnReferenceForward(tokens, nextIdx+1)
}

func parseColumnReferenceBackward(tokens []tokenizer.Token, idx int) (string, string, bool) {
	if idx < 0 || idx >= len(tokens) {
		return "", "", false
	}
	tok := tokens[idx]
	if !isIdentifierToken(tok) {
		return "", "", false
	}
	column := tokenizer.NormalizeIdentifier(tok.Text)
	table := ""
	if idx >= 2 {
		dotTok := tokens[idx-1]
		prevTok := tokens[idx-2]
		if dotTok.Kind == tokenizer.KindSymbol && dotTok.Text == "." && isIdentifierToken(prevTok) {
			table = tokenizer.NormalizeIdentifier(prevTok.Text)
		}
	}
	return table, column, true
}

func parseColumnReferenceForward(tokens []tokenizer.Token, idx int) (string, string, bool) {
	if idx >= len(tokens) {
		return "", "", false
	}
	tok := tokens[idx]
	if !isIdentifierToken(tok) {
		return "", "", false
	}
	table := ""
	column := tokenizer.NormalizeIdentifier(tok.Text)
	if idx+2 < len(tokens) {
		dotTok := tokens[idx+1]
		nextTok := tokens[idx+2]
		if dotTok.Kind == tokenizer.KindSymbol && dotTok.Text == "." && isIdentifierToken(nextTok) {
			table = tokenizer.NormalizeIdentifier(tok.Text)
			column = tokenizer.NormalizeIdentifier(nextTok.Text)
		}
	}
	return table, column, true
}

func isIdentifierToken(tok tokenizer.Token) bool {
	return tok.Kind == tokenizer.KindIdentifier
}

func inferInsertParams(cat *model.Catalog, tokens []tokenizer.Token, paramIndexByToken map[int]int, infos map[int]paramInfo) {
	tableName, columns, paramOrder := parseInsertStructure(tokens, paramIndexByToken)
	if tableName == "" || len(columns) == 0 || len(paramOrder) == 0 {
		return
	}
	table := lookupTable(cat, tableName)
	if table == nil {
		return
	}
	limit := len(columns)
	if len(paramOrder) < limit {
		limit = len(paramOrder)
	}
	for i := 0; i < limit; i++ {
		paramIdx := paramOrder[i]
		if _, exists := infos[paramIdx]; exists {
			continue
		}
		schemaCol := lookupColumn(table, columns[i])
		if schemaCol == nil {
			continue
		}
		infos[paramIdx] = paramInfo{
			GoType:   SQLiteTypeToGo(schemaCol.Type),
			Nullable: !schemaCol.NotNull,
		}
	}
}

func parseInsertStructure(tokens []tokenizer.Token, paramIndexByToken map[int]int) (string, []string, []int) {
	var tableName string
	columns := make([]string, 0, 4)
	params := make([]int, 0, 4)

	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindKeyword && tok.Text == "INTO" {
			i++
			break
		}
		i++
	}
	if i >= len(tokens) {
		return "", nil, nil
	}
	if i < len(tokens) && isIdentifierToken(tokens[i]) {
		tableName = tokenizer.NormalizeIdentifier(tokens[i].Text)
		i++
	} else {
		return "", nil, nil
	}

	if i < len(tokens) && tokens[i].Kind == tokenizer.KindSymbol && tokens[i].Text == "(" {
		i++
		for i < len(tokens) {
			tok := tokens[i]
			if tok.Kind == tokenizer.KindSymbol {
				if tok.Text == ")" {
					i++
					break
				}
				i++
				continue
			}
			if isIdentifierToken(tok) {
				columns = append(columns, tokenizer.NormalizeIdentifier(tok.Text))
			}
			i++
		}
	}

	for i < len(tokens) {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindKeyword && tok.Text == "VALUES" {
			i++
			break
		}
		i++
	}
	if i >= len(tokens) {
		return tableName, columns, nil
	}
	if tokens[i].Kind != tokenizer.KindSymbol || tokens[i].Text != "(" {
		return tableName, columns, nil
	}
	i++
	depth := 1
	for i < len(tokens) && depth > 0 {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				depth--
			case ":":
				if depth == 1 {
					if paramIdx, ok := paramIndexByToken[i]; ok {
						params = append(params, paramIdx)
					}
				}
			case "?":
				if depth == 1 {
					if paramIdx, ok := paramIndexByToken[i]; ok {
						params = append(params, paramIdx)
					}
				}
			}
			i++
			continue
		}
		i++
	}

	return tableName, columns, params
}

func actualTokenLine(q parser.Query, tok tokenizer.Token) int {
	if tok.Line == 0 {
		return q.Block.Line
	}
	return q.Block.Line + tok.Line
}

func SQLiteTypeToGo(sqliteType string) string {
	base := normalizeSQLiteType(sqliteType)
	switch base {
	case "INTEGER":
		return "int64"
	case "REAL":
		return "float64"
	case "TEXT":
		return "string"
	case "BLOB":
		return "[]byte"
	case "NUMERIC":
		return "string"
	default:
		return "interface{}"
	}
}

func normalizeSQLiteType(sqliteType string) string {
	s := strings.TrimSpace(sqliteType)
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	for i, r := range upper {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return upper[:i]
		}
	}
	return upper
}
