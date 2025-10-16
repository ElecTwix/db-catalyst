package analyzer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

type Analyzer struct {
	Catalog *model.Catalog
}

func New(catalog *model.Catalog) *Analyzer {
	return &Analyzer{Catalog: catalog}
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
	Name     string
	Style    parser.ParamStyle
	GoType   string
	Nullable bool
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
	if catalog == nil {
		addDiag(Diagnostic{
			Path:     q.Block.Path,
			Line:     q.Block.Line,
			Column:   q.Block.Column,
			Message:  "schema catalog unavailable; using default column and parameter types",
			Severity: SeverityWarning,
		})
	}

	for _, col := range q.Columns {
		rc := ResultColumn{
			Name:     columnDisplayName(col),
			Table:    col.Table,
			GoType:   "interface{}",
			Nullable: true,
		}

		if catalog != nil && col.Table != "" {
			table := lookupTable(catalog, col.Table)
			if table == nil {
				addDiag(Diagnostic{
					Path:     q.Block.Path,
					Line:     col.Line,
					Column:   col.Column,
					Message:  fmt.Sprintf("result column %q references unknown table %q", rc.Name, col.Table),
					Severity: SeverityError,
				})
			} else {
				columnName := deriveColumnName(col)
				if columnName == "" {
					addDiag(Diagnostic{
						Path:     q.Block.Path,
						Line:     col.Line,
						Column:   col.Column,
						Message:  fmt.Sprintf("result column %q could not determine base column for table %q", rc.Name, col.Table),
						Severity: SeverityError,
					})
				} else if schemaCol := lookupColumn(table, columnName); schemaCol != nil {
					rc.GoType = SQLiteTypeToGo(schemaCol.Type)
					rc.Nullable = !schemaCol.NotNull
				} else {
					addDiag(Diagnostic{
						Path:     q.Block.Path,
						Line:     col.Line,
						Column:   col.Column,
						Message:  fmt.Sprintf("result column %q references unknown column %s.%s", rc.Name, col.Table, columnName),
						Severity: SeverityError,
					})
				}
			}
		} else if catalog != nil && col.Alias != "" {
			addDiag(Diagnostic{
				Path:     q.Block.Path,
				Line:     col.Line,
				Column:   col.Column,
				Message:  fmt.Sprintf("result column %q derives from expression without schema mapping", col.Alias),
				Severity: SeverityWarning,
			})
		}

		result.Columns = append(result.Columns, rc)
	}

	paramInfos := inferParamTypes(catalog, q)
	for idx, param := range q.Params {
		rp := ResultParam{
			Name:     param.Name,
			Style:    param.Style,
			GoType:   "interface{}",
			Nullable: true,
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

type paramInfo struct {
	GoType   string
	Nullable bool
}

type posKey struct {
	line   int
	column int
}

func inferParamTypes(cat *model.Catalog, q parser.Query) map[int]paramInfo {
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
		if !ok || table == "" || column == "" {
			continue
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
