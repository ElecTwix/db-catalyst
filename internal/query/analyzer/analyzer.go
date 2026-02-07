// Package analyzer validates and resolves SQL queries against a schema catalog.
//
//nolint:goconst // SQL keywords and type names are naturally repeated
package analyzer

import (
	"fmt"
	"maps"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Analyzer validates queries against the schema catalog.
type Analyzer struct {
	Catalog      *model.Catalog
	CustomTypes  map[string]config.CustomTypeMapping
	typeResolver TypeResolver
}

// TypeResolver interface for database-specific type mapping.
type TypeResolver interface {
	ResolveType(sqlType string, nullable bool) TypeInfo
}

// TypeInfo describes a resolved Go type.
type TypeInfo struct {
	GoType      string
	UsesSQLNull bool
	Import      string
	Package     string
}

// Result contains the analysis result for a single query.
type Result struct {
	Query       parser.Query
	Columns     []ResultColumn
	Params      []ResultParam
	Diagnostics []Diagnostic
}

// ResultColumn describes a single output column of a query.
type ResultColumn struct {
	Name     string
	Table    string
	GoType   string
	Nullable bool
}

// ResultParam describes a single input parameter of a query.
type ResultParam struct {
	Name          string
	Style         parser.ParamStyle
	GoType        string
	Nullable      bool
	IsVariadic    bool
	VariadicCount int
}

// Diagnostic represents an issue found during analysis.
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// Severity indicates the seriousness of a diagnostic.
type Severity int

const (
	// SeverityWarning indicates a potential issue that doesn't prevent code generation.
	SeverityWarning Severity = iota
	// SeverityError indicates a fatal issue that prevents code generation.
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
	name        string
	columns     []scopeColumn
	columnIndex map[string]int
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

// New creates a new Analyzer with the given catalog.
func New(catalog *model.Catalog) *Analyzer {
	return &Analyzer{Catalog: catalog, CustomTypes: nil}
}

// NewWithCustomTypes creates a new Analyzer with the given catalog and custom type mappings.
func NewWithCustomTypes(catalog *model.Catalog, customTypes map[string]config.CustomTypeMapping) *Analyzer {
	return &Analyzer{Catalog: catalog, CustomTypes: customTypes}
}

// Analyze validates and resolves a parsed query.
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
			entry := a.scopeEntryFromTable(table)
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
		addDiag(Diagnostic{
			Path:     q.Block.Path,
			Line:     q.Block.Line,
			Column:   q.Block.Column,
			Message:  fmt.Sprintf("tokenization failed: %v", err),
			Severity: SeverityError,
		})
		return Result{
			Query:       q,
			Columns:     nil,
			Params:      nil,
			Diagnostics: result.Diagnostics,
		}
	}

	workingScope := newQueryScope()
	if tokens != nil {
		// Populate baseScope with all aliases in the entire block (including CTEs)
		// so that inferParamTypes can resolve columns like u.id correctly.
		addAliasesFromTokens(baseScope, tokens)

		mainIdx := findMainStatementStart(tokens)
		mainTokens := tokens
		if mainIdx >= 0 {
			mainTokens = tokens[mainIdx:]
		}

		// Only add tables/CTEs that are actually referenced in FROM/JOIN/INSERT/UPDATE/DELETE
		referenced := discoverReferencedRelations(mainTokens)
		// Always include CTEs in the working scope if they are referenced
		for _, cte := range q.CTEs {
			referenced = append(referenced, cte.Name)
		}

		for _, ref := range referenced {
			if entry, ok := baseScope.get(ref); ok {
				workingScope.addEntry(ref, entry)
			}
		}
		addAliasesFromTokens(workingScope, mainTokens)
	} else {
		workingScope = baseScope.clone()
	}

	for _, col := range q.Columns {
		if col.Expr == "*" || strings.HasSuffix(col.Expr, ".*") {
			expanded, diags := expandStar(col, workingScope, q.Block, hasCatalog)
			result.Columns = append(result.Columns, expanded...)
			for _, d := range diags {
				addDiag(d)
			}
			continue
		}

		rc, diags := resolveResultColumn(col, workingScope, q.Block, hasCatalog)
		result.Columns = append(result.Columns, rc)
		for _, d := range diags {
			addDiag(d)
		}
	}

	// Handle RETURNING clause for DML statements
	if (q.Verb == parser.VerbInsert || q.Verb == parser.VerbUpdate || q.Verb == parser.VerbDelete) && tokens != nil {
		returningCols, diags := discoverReturningColumns(tokens, q.Block, workingScope, hasCatalog)
		result.Columns = append(result.Columns, returningCols...)
		for _, d := range diags {
			addDiag(d)
		}
	}

	paramInfos := a.inferParamTypes(q, workingScope, baseScope)

	// Build a map of explicit type overrides from block annotations
	explicitTypes := make(map[string]paramInfo)
	for _, pt := range q.Block.ParamTypes {
		explicitTypes[pt.ParamName] = paramInfo{GoType: pt.GoType, Nullable: false}
	}

	for idx, param := range q.Params {
		rp := ResultParam{
			Name:          param.Name,
			Style:         param.Style,
			GoType:        "any",
			Nullable:      true,
			IsVariadic:    param.IsVariadic,
			VariadicCount: param.VariadicCount,
		}

		// Check for explicit type override first
		if info, ok := explicitTypes[param.Name]; ok {
			rp.GoType = info.GoType
			rp.Nullable = info.Nullable
		} else if info, ok := paramInfos[idx]; ok {
			rp.GoType = info.GoType
			rp.Nullable = info.Nullable
		}
		result.Params = append(result.Params, rp)
	}

	// Validate all identifiers in the main statement (WHERE, ORDER BY, etc.)
	if tokens != nil && hasCatalog {
		mainIdx := findMainStatementStart(tokens)
		if mainIdx >= 0 {
			diags := a.validateIdentifiers(tokens[mainIdx:], workingScope, q.Block)
			for _, d := range diags {
				addDiag(d)
			}
		}
	}

	return result
}

func findMainStatementStart(tokens []tokenizer.Token) int {
	depth := 0
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
		if depth == 0 && (tok.Kind == tokenizer.KindKeyword || tok.Kind == tokenizer.KindIdentifier) {
			upper := strings.ToUpper(tok.Text)
			switch upper {
			case "SELECT", "INSERT", "UPDATE", "DELETE":
				return i
			}
		}
	}
	return -1
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

	workingScope := newQueryScope()
	anchorTokens, err := tokenizer.Scan(parent.Block.Path, []byte(anchorSQL), false)
	if err == nil {
		referenced := discoverReferencedRelations(anchorTokens)
		for _, ref := range referenced {
			if entry, ok := scope.get(ref); ok {
				workingScope.addEntry(ref, entry)
			}
		}
		addAliasesFromTokens(workingScope, anchorTokens)
	} else {
		workingScope = scope.clone()
	}

	columnNames, colDiags := cteOutputNames(cte, anchorQuery)
	diags = append(diags, colDiags...)
	if len(columnNames) == 0 {
		return nil, diags
	}

	resolved := make([]scopeColumn, 0, len(columnNames))
	for idx, col := range anchorQuery.Columns {
		name := columnNames[idx]
		sc := scopeColumn{name: name, owner: cte.Name, goType: "any", nullable: true}

		suppressDefaultWarning := false
		if hasCatalog {
			colInfo, warn := a.resolveCTEColumn(col, cte, workingScope, parent.Block.Path)
			sc.goType = colInfo.goType
			sc.nullable = colInfo.nullable
			suppressDefaultWarning = colInfo.suppressWarning
			diags = append(diags, warn...)
		}

		if sc.goType == "any" && hasCatalog && !suppressDefaultWarning {
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
		name:        cte.Name,
		columns:     make([]scopeColumn, 0, len(resolved)),
		columnIndex: make(map[string]int, len(resolved)),
	}
	for _, col := range resolved {
		idx := len(entry.columns)
		entry.columns = append(entry.columns, col)
		entry.columnIndex[normalizeIdent(col.name)] = idx
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

// cteColumnInfo holds resolved information for a CTE column.
type cteColumnInfo struct {
	goType          string
	nullable        bool
	suppressWarning bool
}

// resolveCTEColumn resolves a single CTE column's type information.
func (a *Analyzer) resolveCTEColumn(col parser.Column, cte parser.CTE, workingScope *queryScope, path string) (cteColumnInfo, []Diagnostic) {
	info := cteColumnInfo{goType: "any", nullable: true}
	var diags []Diagnostic

	alias := col.Table
	columnName := deriveColumnName(col)
	agg, isAggregate := parseAggregateExpr(col.Expr)
	skipLookup := false

	if isAggregate {
		switch {
		case agg.argStar:
			info.goType = "int64"
			info.nullable = false
			skipLookup = true
		case agg.argColumn == "":
			diags = append(diags, Diagnostic{
				Path:     path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("unable to infer metadata for aggregate %s in CTE %s; defaulting to interface{}", col.Expr, cte.Name),
				Severity: SeverityWarning,
			})
			info.suppressWarning = true
			skipLookup = true
		default:
			alias = agg.argAlias
			columnName = agg.argColumn
		}
	}

	if skipLookup {
		return info, diags
	}

	lookup, _, res := workingScope.lookup(alias, columnName)
	switch res {
	case scopeLookupOK:
		if isAggregate {
			info = a.resolveAggregateType(agg, lookup, col, cte, path)
		} else {
			info.goType = lookup.goType
			info.nullable = lookup.nullable
		}
	case scopeLookupAliasNotFound:
		diags = append(diags, a.makeAliasNotFoundDiag(alias, col, cte, path)...)
	case scopeLookupColumnNotFound:
		diags = append(diags, a.makeColumnNotFoundDiag(alias, columnName, col, cte, path)...)
	case scopeLookupAmbiguous:
		diags = append(diags, Diagnostic{
			Path:     path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("CTE %s column %s is ambiguous; qualify with a table alias", cte.Name, columnName),
			Severity: SeverityError,
		})
	}

	return info, diags
}

// resolveAggregateType determines the type for an aggregate expression.
func (a *Analyzer) resolveAggregateType(agg aggregateExpr, lookup scopeColumn, _ parser.Column, _ parser.CTE, _ string) cteColumnInfo {
	info := cteColumnInfo{goType: "any", nullable: true, suppressWarning: true}

	goType, nullable, ok := aggregateResultFromOperand(agg.kind, lookup)
	if !ok {
		return info
	}

	info.goType = goType
	info.nullable = nullable
	info.suppressWarning = false
	return info
}

// makeAliasNotFoundDiag creates diagnostics for alias not found errors.
func (a *Analyzer) makeAliasNotFoundDiag(alias string, col parser.Column, cte parser.CTE, path string) []Diagnostic {
	var diags []Diagnostic
	if alias != "" {
		diags = append(diags, Diagnostic{
			Path:     path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("CTE %s references unknown relation %s", cte.Name, alias),
			Severity: SeverityError,
		})
	} else if col.Alias != "" {
		diags = append(diags, Diagnostic{
			Path:     path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("CTE %s column %s derives from expression without schema mapping", cte.Name, col.Alias),
			Severity: SeverityWarning,
		})
	}
	return diags
}

// makeColumnNotFoundDiag creates diagnostics for column not found errors.
func (a *Analyzer) makeColumnNotFoundDiag(alias, columnName string, col parser.Column, cte parser.CTE, path string) []Diagnostic {
	var diags []Diagnostic
	if alias != "" {
		diags = append(diags, Diagnostic{
			Path:     path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("CTE %s references unknown column %s.%s", cte.Name, alias, columnName),
			Severity: SeverityError,
		})
	} else if col.Alias != "" {
		diags = append(diags, Diagnostic{
			Path:     path,
			Line:     col.Line,
			Column:   col.Column,
			Message:  fmt.Sprintf("CTE %s column %s derives from expression without schema mapping", cte.Name, col.Alias),
			Severity: SeverityWarning,
		})
	}
	return diags
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
	aggregateKindCoalesce
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
	var kind aggregateKind
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
	case "COALESCE":
		kind = aggregateKindCoalesce
	default:
		return aggregateExpr{}, false
	}

	closeIdx := strings.LastIndex(trimmed, ")")
	if closeIdx < 0 || closeIdx <= open {
		return aggregateExpr{}, false
	}
	after := strings.TrimSpace(trimmed[closeIdx+1:])
	if after != "" {
		return aggregateExpr{}, false
	}

	inner := strings.TrimSpace(trimmed[open+1 : closeIdx])
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

	// For COALESCE, we just take the first argument for the default name if it's a simple column
	if kind == aggregateKindCoalesce {
		parts := strings.Split(inner, ",")
		if len(parts) > 0 {
			inner = strings.TrimSpace(parts[0])
		}
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
	case aggregateKindCoalesce:
		return "COALESCE"
	default:
		return "aggregate"
	}
}

func defaultAggregateName(agg aggregateExpr) string {
	var base string
	switch {
	case agg.argStar:
		base = "count"
	case agg.argColumn != "":
		base = agg.argColumn
	default:
		base = "column"
	}

	switch agg.kind {
	case aggregateKindCount:
		if agg.argStar {
			return "count"
		}
		return "count_" + base
	case aggregateKindSum:
		return "sum_" + base
	case aggregateKindMin:
		return "min_" + base
	case aggregateKindMax:
		return "max_" + base
	case aggregateKindAvg:
		return "avg_" + base
	case aggregateKindCoalesce:
		return base
	default:
		return base
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
	case aggregateKindMin, aggregateKindMax, aggregateKindCoalesce:
		if operand.goType == "" || operand.goType == "any" {
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

func expandStar(col parser.Column, scope *queryScope, blk block.Block, hasCatalog bool) ([]ResultColumn, []Diagnostic) {
	if !hasCatalog || scope == nil {
		return []ResultColumn{{
			Name:     "*",
			GoType:   "any",
			Nullable: true,
		}}, nil
	}

	alias := col.Table
	if col.Expr != "*" {
		// handle table.*
		if dot := strings.LastIndex(col.Expr, "."); dot >= 0 {
			alias = tokenizer.NormalizeIdentifier(col.Expr[:dot])
		}
	}

	if alias != "" {
		entry, ok := scope.get(alias)
		if !ok {
			return nil, []Diagnostic{{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("star expansion references unknown table %q", alias),
				Severity: SeverityError,
			}}
		}
		return entryToResultColumns(entry), nil
	}

	// Expand all tables in scope
	var cols []ResultColumn
	seen := make(map[*scopeEntry]struct{})
	for _, entry := range scope.entries {
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		cols = append(cols, entryToResultColumns(entry)...)
	}
	return cols, nil
}

func entryToResultColumns(entry *scopeEntry) []ResultColumn {
	cols := make([]ResultColumn, 0, len(entry.columns))
	// Columns are stored in schema order to ensure deterministic output
	for _, sc := range entry.columns {
		cols = append(cols, ResultColumn{
			Name:     sc.name,
			Table:    entry.name,
			GoType:   sc.goType,
			Nullable: sc.nullable,
		})
	}
	return cols
}

func discoverReturningColumns(tokens []tokenizer.Token, blk block.Block, scope *queryScope, hasCatalog bool) ([]ResultColumn, []Diagnostic) {
	returningIdx := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Kind == tokenizer.KindKeyword && strings.ToUpper(tokens[i].Text) == "RETURNING" {
			returningIdx = i
			break
		}
	}

	if returningIdx == -1 {
		return nil, nil
	}

	// This is a simplified version of parseSelectColumns but for tokens after RETURNING
	// In a real implementation, we should probably let the parser handle RETURNING clauses.
	var cols []ResultColumn
	var diags []Diagnostic

	// For now, we only support RETURNING * or RETURNING col1, col2
	i := returningIdx + 1
	for i < len(tokens) {
		if tokens[i].Kind == tokenizer.KindEOF {
			break
		}
		if tokens[i].Kind == tokenizer.KindSymbol && tokens[i].Text == "," {
			i++
			continue
		}
		if tokens[i].Kind == tokenizer.KindSymbol && tokens[i].Text == "*" {
			expanded, eDiags := expandStar(parser.Column{Expr: "*", Line: tokens[i].Line, Column: tokens[i].Column}, scope, blk, hasCatalog)
			cols = append(cols, expanded...)
			diags = append(diags, eDiags...)
			i++
			continue
		}
		// Handle simple identifier or table.identifier
		if tokens[i].Kind == tokenizer.KindIdentifier {
			// Basic lookahead for alias: col AS alias
			name := tokenizer.NormalizeIdentifier(tokens[i].Text)
			alias := ""
			next := i + 1
			if next < len(tokens) && tokens[next].Kind == tokenizer.KindKeyword && strings.ToUpper(tokens[next].Text) == "AS" {
				next++
			}
			if next < len(tokens) && tokens[next].Kind == tokenizer.KindIdentifier {
				alias = tokenizer.NormalizeIdentifier(tokens[next].Text)
				i = next
			}

			rc, rDiags := resolveResultColumn(parser.Column{Expr: name, Alias: alias, Line: tokens[i].Line, Column: tokens[i].Column}, scope, blk, hasCatalog)
			cols = append(cols, rc)
			diags = append(diags, rDiags...)
		}
		i++
	}

	return cols, diags
}

func resolveResultColumn(col parser.Column, scope *queryScope, blk block.Block, hasCatalog bool) (ResultColumn, []Diagnostic) {
	rc := ResultColumn{
		Name:     columnDisplayName(col),
		Table:    col.Table,
		GoType:   "any",
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
			rc.Name = defaultAggregateName(agg)
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("aggregate %s requires an alias; defaulting to %q", aggregateKindString(agg.kind), rc.Name),
				Severity: SeverityWarning,
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
				rc.GoType = "any"
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
		switch {
		case alias != "":
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q references unknown column %s.%s", rcOrExprName(rc, col), alias, columnName),
				Severity: SeverityError,
			})
		case col.Alias != "":
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q derives from expression without schema mapping", col.Alias),
				Severity: SeverityWarning,
			})
		default:
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
	maps.Copy(clone.entries, s.entries)
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
		idx, found := entry.columnIndex[normalizeIdent(column)]
		if !found {
			return scopeColumn{}, entry, scopeLookupColumnNotFound
		}
		return entry.columns[idx], entry, scopeLookupOK
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
		idx, ok := entry.columnIndex[normalizeIdent(column)]
		if ok {
			foundCol = entry.columns[idx]
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

func (a *Analyzer) scopeEntryFromTable(tbl *model.Table) *scopeEntry {
	cols := make([]scopeColumn, 0, len(tbl.Columns))
	colIndex := make(map[string]int, len(tbl.Columns))
	for _, col := range tbl.Columns {
		idx := len(cols)
		cols = append(cols, scopeColumn{
			name:     col.Name,
			owner:    tbl.Name,
			goType:   a.SQLiteTypeToGo(col.Type),
			nullable: !col.NotNull,
		})
		colIndex[normalizeIdent(col.Name)] = idx
	}
	return &scopeEntry{name: tbl.Name, columns: cols, columnIndex: colIndex}
}

func normalizeIdent(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(tokenizer.NormalizeIdentifier(name))
}

func shouldSkipIdentifier(isQualified bool, table, upperName, name string, scope *queryScope, queryAliases map[string]struct{}) bool {
	if isQualified {
		// Skip sqlc macros: sqlc.arg, sqlc.narg, sqlc.slice
		if strings.ToUpper(table) == "SQLC" {
			return true
		}
		return false
	}
	// Skip common SQL functions and special identifiers for bare names
	if isCommonFunction(upperName) || upperName == "SQLC" {
		return true
	}

	// Skip if it's a known relation name (table or CTE)
	if _, ok := scope.get(name); ok {
		return true
	}

	// Skip if it's an alias defined in this query
	if _, ok := queryAliases[normalizeIdent(name)]; ok {
		return true
	}
	return false
}

func (a *Analyzer) validateIdentifiers(tokens []tokenizer.Token, scope *queryScope, blk block.Block) []Diagnostic {
	var diags []Diagnostic

	// Collect aliases from the current query to avoid false positives for column aliases in ORDER BY etc.
	queryAliases := make(map[string]struct{})
	q, _ := parser.Parse(blk)
	for _, col := range q.Columns {
		if col.Alias != "" {
			queryAliases[normalizeIdent(col.Alias)] = struct{}{}
		}
	}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind != tokenizer.KindIdentifier {
			continue
		}

		// Skip if it's a keyword (case-insensitive)
		if tokenizer.IsKeyword(tok.Text) {
			continue
		}

		name := tokenizer.NormalizeIdentifier(tok.Text)
		upperName := strings.ToUpper(name)

		// Check for qualified identifier: table.column
		table := ""
		column := name
		isQualified := false
		if i+2 < len(tokens) && tokens[i+1].Kind == tokenizer.KindSymbol && tokens[i+1].Text == "." {
			if tokens[i+2].Kind == tokenizer.KindIdentifier || (tokens[i+2].Kind == tokenizer.KindSymbol && tokens[i+2].Text == "*") {
				table = name
				column = tokenizer.NormalizeIdentifier(tokens[i+2].Text)
				isQualified = true
				i += 2
			}
		}

		if shouldSkipIdentifier(isQualified, table, upperName, name, scope, queryAliases) {
			continue
		}

		// If it's a star, it's already handled or valid in some contexts
		if column == "*" {
			continue
		}

		// Validate the identifier against the scope
		_, _, res := scope.lookup(table, column)
		switch res {
		case scopeLookupAliasNotFound:
			if table != "" {
				diags = append(diags, Diagnostic{
					Path:     blk.Path,
					Line:     tok.Line,
					Column:   tok.Column,
					Message:  fmt.Sprintf("unknown table/alias %q", table),
					Severity: SeverityError,
				})
			}
		case scopeLookupColumnNotFound:
			if table != "" {
				diags = append(diags, Diagnostic{
					Path:     blk.Path,
					Line:     tok.Line,
					Column:   tok.Column,
					Message:  fmt.Sprintf("unknown column %q", column),
					Severity: SeverityError,
				})
			}
		case scopeLookupAmbiguous:
			diags = append(diags, Diagnostic{
				Path:     blk.Path,
				Line:     tok.Line,
				Column:   tok.Column,
				Message:  fmt.Sprintf("ambiguous column %q; qualify with a table alias", column),
				Severity: SeverityError,
			})
		}
	}
	return diags
}

func isCommonFunction(name string) bool {
	switch name {
	case "COUNT", "SUM", "MIN", "MAX", "AVG", "COALESCE", "IFNULL", "NULLIF", "ABS", "LOWER", "UPPER", "TRIM", "LENGTH", "RANDOM", "ROUND", "REPLACE", "SUBSTR", "DATE", "TIME", "DATETIME", "STRFTIME", "UNIXEPOCH", "CAST", "EXISTS":
		return true
	default:
		return false
	}
}

func discoverReferencedRelations(tokens []tokenizer.Token) []string {
	var referenced []string
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		if tok.Kind != tokenizer.KindKeyword {
			i++
			continue
		}
		text := strings.ToUpper(tok.Text)
		if text != "FROM" && text != "JOIN" && text != "INTO" && text != "UPDATE" && text != "DELETE" {
			i++
			continue
		}
		i++
		// For DELETE, we skip FROM if present
		if text == "DELETE" && i < len(tokens) && strings.ToUpper(tokens[i].Text) == "FROM" {
			i++
		}
		relation, ok := parseRelationName(tokens, &i)
		if ok {
			referenced = append(referenced, relation)
		}
	}
	return referenced
}

func addAliasesFromTokens(scope *queryScope, tokens []tokenizer.Token) {
	if scope == nil || len(tokens) == 0 {
		return
	}
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindEOF {
			break
		}
		if tok.Kind != tokenizer.KindKeyword {
			i++
			continue
		}
		text := strings.ToUpper(tok.Text)
		if text != "FROM" && text != "JOIN" {
			i++
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

func (a *Analyzer) inferParamTypes(q parser.Query, scope *queryScope, baseScope *queryScope) map[int]paramInfo {
	cat := a.Catalog
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

		isSlice := q.Params[paramIdx].IsVariadic

		var table, column string
		var ok bool

		if isSlice {
			table, column, ok = matchInReference(tokens, tokenIdx)
		}

		if !ok {
			table, column, ok = matchEqualityReference(tokens, tokenIdx)
		}

		if !ok {
			continue
		}

		var typeName string
		var nullable bool
		found := false

		if scope != nil {
			if resolved, _, status := scope.lookup(table, column); status == scopeLookupOK && resolved.goType != "any" {
				typeName = resolved.goType
				nullable = resolved.nullable
				found = true
			}
		}

		if !found && baseScope != nil {
			if resolved, _, status := baseScope.lookup(table, column); status == scopeLookupOK && resolved.goType != "any" {
				typeName = resolved.goType
				nullable = resolved.nullable
				found = true
			} else if (status == scopeLookupAliasNotFound || status == scopeLookupAmbiguous) && column != "" {
				// Final fallback: try global lookup in baseScope if alias not found or ambiguous
				if fallback, _, fbStatus := baseScope.lookup("", column); fbStatus == scopeLookupOK && fallback.goType != "any" {
					typeName = fallback.goType
					nullable = fallback.nullable
					found = true
				}
			}
		}

		if !found {
			if info, schemaFound := a.schemaInfoForColumn(cat, table, column); schemaFound {
				typeName = info.GoType
				nullable = info.Nullable
				found = true
			}
		}

		if found {
			if isSlice {
				// For slices, we want the base type wrapped in a slice, and typically not nullable elements
				// unless the column is nullable. But usually input slices are []Type.
				// If the column is nullable, sqlc usually generates []Type (and expects no nulls or handles them).
				// We'll stick to non-pointer slice elements for now as that's typical for IN clauses.
				typeName = "[]" + typeName
				nullable = false // Slices themselves aren't nullable in this context usually
			}
			infos[paramIdx] = paramInfo{GoType: typeName, Nullable: nullable}
		}
	}

	if q.Verb == parser.VerbInsert {
		a.inferInsertParams(cat, tokens, paramIndexByToken, infos)
	}

	if len(infos) == 0 {
		return nil
	}
	return infos
}

func (a *Analyzer) schemaInfoForColumn(cat *model.Catalog, tableName, columnName string) (paramInfo, bool) {
	table := lookupTable(cat, tableName)
	if table == nil {
		return paramInfo{}, false
	}
	column := lookupColumn(table, columnName)
	if column == nil {
		return paramInfo{}, false
	}
	return paramInfo{
		GoType:   a.SQLiteTypeToGo(column.Type),
		Nullable: !column.NotNull,
	}, true
}

func matchInReference(tokens []tokenizer.Token, paramIdx int) (string, string, bool) {
	const minTokensForIN = 2
	if paramIdx < minTokensForIN {
		return "", "", false
	}
	// Looking for: col IN ( ? )
	// paramIdx points to ?, so paramIdx-1 should be (, paramIdx-2 should be IN
	if tokens[paramIdx-1].Kind != tokenizer.KindSymbol || tokens[paramIdx-1].Text != "(" {
		return "", "", false
	}
	if tokens[paramIdx-2].Kind != tokenizer.KindKeyword || strings.ToUpper(tokens[paramIdx-2].Text) != "IN" {
		// IN is a keyword.
		// Check if tokenizer marked it as Identifier?
		if strings.ToUpper(tokens[paramIdx-2].Text) != "IN" {
			return "", "", false
		}
	}
	// Check if IN is preceded by NOT
	idx := paramIdx - 3 //nolint:mnd // looking 3 tokens back for column reference
	if idx >= 0 && tokens[idx].Kind == tokenizer.KindKeyword && strings.ToUpper(tokens[idx].Text) == "NOT" {
		idx--
	}

	return parseColumnReferenceBackward(tokens, idx)
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

func (a *Analyzer) inferInsertParams(cat *model.Catalog, tokens []tokenizer.Token, paramIndexByToken map[int]int, infos map[int]paramInfo) {
	tableName, columns, paramOrder := parseInsertStructure(tokens, paramIndexByToken)
	if tableName == "" || len(columns) == 0 || len(paramOrder) == 0 {
		return
	}
	table := lookupTable(cat, tableName)
	if table == nil {
		return
	}
	limit := min(len(columns), len(paramOrder))
	for i := range limit {
		paramIdx := paramOrder[i]
		if _, exists := infos[paramIdx]; exists {
			continue
		}
		schemaCol := lookupColumn(table, columns[i])
		if schemaCol == nil {
			continue
		}
		goType := a.SQLiteTypeToGo(schemaCol.Type)
		infos[paramIdx] = paramInfo{
			GoType:   goType,
			Nullable: !schemaCol.NotNull,
		}
	}
}

func parseInsertStructure(tokens []tokenizer.Token, paramIndexByToken map[int]int) (string, []string, []int) {
	var tableName string
	columns := make([]string, 0, 4) //nolint:mnd // typical column count

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
	params := collectInsertParams(tokens, i, paramIndexByToken)

	return tableName, columns, params
}

func collectInsertParams(tokens []tokenizer.Token, start int, paramIndexByToken map[int]int) []int {
	var params []int
	depth := 1
	i := start
	for i < len(tokens) && depth > 0 {
		tok := tokens[i]
		// Handle both symbol tokens (parentheses) and param tokens ($N)
		switch tok.Kind {
		case tokenizer.KindSymbol:
			depth = updateDepth(tok.Text, depth)
			params = maybeCollectParam(tok.Text, depth, i, paramIndexByToken, params)
		case tokenizer.KindParam:
			// PostgreSQL-style parameter at depth 1 (inside VALUES but not nested)
			if depth == 1 {
				if paramIdx, ok := paramIndexByToken[i]; ok {
					params = append(params, paramIdx)
				}
			}
		}
		i++
	}
	return params
}

// updateDepth adjusts the parenthesis depth based on the token.
func updateDepth(text string, depth int) int {
	switch text {
	case "(":
		return depth + 1
	case ")":
		return depth - 1
	}
	return depth
}

// maybeCollectParam adds a parameter index if the token is a parameter at depth 1.
func maybeCollectParam(text string, depth, idx int, paramIndexByToken map[int]int, params []int) []int {
	if depth != 1 {
		return params
	}
	if text != ":" && text != "?" {
		return params
	}
	if paramIdx, ok := paramIndexByToken[idx]; ok {
		params = append(params, paramIdx)
	}
	return params
}

func actualTokenLine(q parser.Query, tok tokenizer.Token) int {
	if tok.Line == 0 {
		return q.Block.Line
	}
	return q.Block.Line + tok.Line
}

// SQLiteTypeToGo converts a SQLite type to a Go type.
// If a TypeResolver is set, it will be used for database-specific type mapping.
func (a *Analyzer) SQLiteTypeToGo(sqliteType string) string {
	// Use TypeResolver if available (for PostgreSQL, MySQL, etc.)
	if a.typeResolver != nil {
		typeInfo := a.typeResolver.ResolveType(sqliteType, false)
		if typeInfo.GoType != "" && typeInfo.GoType != "any" {
			return typeInfo.GoType
		}
	}

	// First check if the SQLite type has a custom type mapping
	if a.CustomTypes != nil {
		normalizedType := normalizeSQLiteType(sqliteType)
		if mapping, exists := a.CustomTypes[normalizedType]; exists {
			// Return the base Go type from custom type
			// Note: pointer logic is handled separately by the TypeResolver during code generation
			return mapping.GoType
		}
	}

	// Fall back to default SQLite type mapping
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
		return "any"
	}
}

// SetTypeResolver sets the type resolver for database-specific type mapping.
func (a *Analyzer) SetTypeResolver(resolver TypeResolver) {
	a.typeResolver = resolver
}

// SQLiteTypeToGo is a convenience function that uses default type mapping
// (kept for backward compatibility where no custom types are needed)
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
		return "any"
	}
}

func normalizeSQLiteType(sqliteType string) string {
	s := strings.TrimSpace(sqliteType)
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	for i, r := range upper {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return upper[:i]
		}
	}
	return upper
}
