package parser

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

type Query struct {
	Block       block.Block
	Verb        Verb
	Columns     []Column
	Params      []Param
	CTEs        []CTE
	Diagnostics []Diagnostic
}

type Verb int

const (
	VerbUnknown Verb = iota
	VerbSelect
	VerbInsert
	VerbUpdate
	VerbDelete
)

type Column struct {
	Expr        string
	Alias       string
	Table       string
	Line        int
	Column      int
	StartOffset int
	EndOffset   int
}

type CTE struct {
	Name      string
	Columns   []string
	SelectSQL string
	Line      int
	Column    int
}

type Param struct {
	Name          string
	Style         ParamStyle
	Order         int
	Line          int
	Column        int
	IsVariadic    bool
	VariadicCount int
}

type ParamStyle int

const (
	ParamStyleUnknown ParamStyle = iota
	ParamStylePositional
	ParamStyleNamed
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

func Parse(blk block.Block) (Query, []Diagnostic) {
	q := Query{Block: blk}
	var diags []Diagnostic

	trimmed := strings.TrimSpace(blk.SQL)
	if trimmed == "" {
		diags = append(diags, Diagnostic{
			Path:     blk.Path,
			Line:     blk.Line,
			Column:   blk.Column,
			Message:  "query block contains no SQL",
			Severity: SeverityError,
		})
		return finalizeQuery(q, diags)
	}

	tokens, err := tokenizer.Scan(blk.Path, []byte(blk.SQL), false)
	if err != nil {
		diags = append(diags, diagnosticFromError(blk, err))
		return finalizeQuery(q, diags)
	}

	posIdx := newPositionIndex(blk.SQL)
	verb, verbIdx, ctes, verbDiags := determineVerb(tokens, blk, posIdx)
	diags = append(diags, verbDiags...)
	if verb == VerbUnknown {
		if len(verbDiags) == 0 {
			if verbIdx >= 0 && verbIdx < len(tokens) {
				tok := tokens[verbIdx]
				diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "unsupported query verb %s", tok.Text))
			} else {
				diags = append(diags, Diagnostic{
					Path:     blk.Path,
					Line:     blk.Line,
					Column:   blk.Column,
					Message:  "could not determine query verb",
					Severity: SeverityError,
				})
			}
		}
		return finalizeQuery(q, diags)
	}
	q.Verb = verb
	if len(ctes) > 0 {
		q.CTEs = ctes
	}

	params, paramDiags := collectParams(tokens, blk)
	q.Params = params
	diags = append(diags, paramDiags...)

	if verb == VerbSelect {
		columns, columnDiags := parseSelectColumns(tokens, verbIdx, blk, posIdx)
		q.Columns = columns
		diags = append(diags, columnDiags...)
	}

	return finalizeQuery(q, diags)
}

func finalizeQuery(q Query, diags []Diagnostic) (Query, []Diagnostic) {
	if len(diags) > 0 {
		q.Diagnostics = append(q.Diagnostics[:0], diags...)
	} else {
		q.Diagnostics = nil
	}
	return q, diags
}

func diagnosticFromError(blk block.Block, err error) Diagnostic {

	if tokErr, ok := err.(*tokenizer.Error); ok {
		return makeDiag(blk, tokErr.Line, tokErr.Column, SeverityError, "%s", tokErr.Message)
	}
	return Diagnostic{
		Path:     blk.Path,
		Line:     blk.Line,
		Column:   blk.Column,
		Message:  err.Error(),
		Severity: SeverityError,
	}
}

func determineVerb(tokens []tokenizer.Token, blk block.Block, pos positionIndex) (Verb, int, []CTE, []Diagnostic) {
	ctes := make([]CTE, 0, 2)
	diags := make([]Diagnostic, 0, 2)

	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindEOF {
			break
		}
		if tok.Kind == tokenizer.KindDocComment {
			i++
			continue
		}
		if tok.Kind != tokenizer.KindKeyword && tok.Kind != tokenizer.KindIdentifier {
			i++
			continue
		}
		text := strings.ToUpper(tok.Text)
		if text == "WITH" {
			list, nextIdx, listDiags := parseCTEList(tokens, i+1, blk, pos)
			if len(list) > 0 {
				ctes = append(ctes, list...)
			}
			if len(listDiags) > 0 {
				diags = append(diags, listDiags...)
				return VerbUnknown, -1, ctes, diags
			}
			i = nextIdx
			continue
		}
		switch text {
		case "SELECT":
			return VerbSelect, i, ctes, diags
		case "INSERT":
			return VerbInsert, i, ctes, diags
		case "UPDATE":
			return VerbUpdate, i, ctes, diags
		case "DELETE":
			return VerbDelete, i, ctes, diags
		}
		if tok.Kind == tokenizer.KindKeyword {
			return VerbUnknown, i, ctes, diags
		}
		i++
	}

	return VerbUnknown, -1, ctes, diags
}

func parseCTEList(tokens []tokenizer.Token, idx int, blk block.Block, pos positionIndex) ([]CTE, int, []Diagnostic) {
	ctes := make([]CTE, 0, 2)
	diags := make([]Diagnostic, 0, 2)
	i := idx

	for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
		i++
	}

	if i < len(tokens) && strings.ToUpper(tokens[i].Text) == "RECURSIVE" {
		i++
	}

	for {
		for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
			i++
		}
		if i >= len(tokens) {
			if idx > 0 && idx-1 < len(tokens) {
				withTok := tokens[idx-1]
				diags = append(diags, makeDiag(blk, withTok.Line, withTok.Column, SeverityError, "WITH clause missing CTE definition"))
			}
			return ctes, i, diags
		}

		nameTok := tokens[i]
		if nameTok.Kind != tokenizer.KindIdentifier {
			diags = append(diags, makeDiag(blk, nameTok.Line, nameTok.Column, SeverityError, "expected CTE name"))
			return ctes, i, diags
		}
		cte := CTE{
			Name: tokenizer.NormalizeIdentifier(nameTok.Text),
		}
		cte.Line, cte.Column = actualPosition(blk, nameTok.Line, nameTok.Column)
		i++

		if i < len(tokens) && tokens[i].Kind == tokenizer.KindSymbol && tokens[i].Text == "(" {
			i++
			columns := make([]string, 0, 4)
			for {
				if i >= len(tokens) {
					diags = append(diags, makeDiag(blk, nameTok.Line, nameTok.Column, SeverityError, "unterminated column list for CTE %s", cte.Name))
					return ctes, i, diags
				}
				tok := tokens[i]
				if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
					i++
					break
				}
				if tok.Kind == tokenizer.KindSymbol && tok.Text == "," {
					i++
					continue
				}
				if tok.Kind != tokenizer.KindIdentifier {
					diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "expected column name in CTE %s", cte.Name))
					return ctes, i, diags
				}
				columns = append(columns, tokenizer.NormalizeIdentifier(tok.Text))
				i++
			}
			cte.Columns = columns
		}

		for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
			i++
		}
		if i >= len(tokens) {
			diags = append(diags, makeDiag(blk, nameTok.Line, nameTok.Column, SeverityError, "expected AS for CTE %s", cte.Name))
			return ctes, i, diags
		}
		asTok := tokens[i]
		if strings.ToUpper(asTok.Text) != "AS" {
			diags = append(diags, makeDiag(blk, asTok.Line, asTok.Column, SeverityError, "expected AS for CTE %s", cte.Name))
			return ctes, i, diags
		}
		i++

		for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
			i++
		}
		if i >= len(tokens) {
			diags = append(diags, makeDiag(blk, asTok.Line, asTok.Column, SeverityError, "missing CTE body for %s", cte.Name))
			return ctes, i, diags
		}
		openTok := tokens[i]
		if openTok.Kind != tokenizer.KindSymbol || openTok.Text != "(" {
			diags = append(diags, makeDiag(blk, openTok.Line, openTok.Column, SeverityError, "expected ( to start CTE %s", cte.Name))
			return ctes, i, diags
		}
		i++
		bodyStart := i
		depth := 1
		bodyEnd := -1
		hasSelect := false

	bodyLoop:
		for i < len(tokens) {
			tok := tokens[i]
			if strings.ToUpper(tok.Text) == "SELECT" {
				hasSelect = true
			}
			if tok.Kind == tokenizer.KindSymbol {
				switch tok.Text {
				case "(":
					depth++
				case ")":
					depth--
					if depth == 0 {
						bodyEnd = i
						break bodyLoop
					}
				}
			}
			i++
		}

		if bodyEnd == -1 {
			diags = append(diags, makeDiag(blk, openTok.Line, openTok.Column, SeverityError, "unterminated CTE %s definition", cte.Name))
			return ctes, i, diags
		}
		if !hasSelect {
			diags = append(diags, makeDiag(blk, openTok.Line, openTok.Column, SeverityError, "CTE %s must contain a SELECT", cte.Name))
			return ctes, i, diags
		}
		if bodyEnd <= bodyStart {
			diags = append(diags, makeDiag(blk, openTok.Line, openTok.Column, SeverityError, "CTE %s must contain a SELECT", cte.Name))
			return ctes, i, diags
		}

		inner := tokens[bodyStart:bodyEnd]
		startTok := inner[0]
		endTok := inner[len(inner)-1]
		startOffset := pos.offset(startTok)
		endOffset := pos.offset(endTok) + len(endTok.Text)
		if endOffset > len(pos.sql) {
			endOffset = len(pos.sql)
		}
		cte.SelectSQL = strings.TrimSpace(pos.sql[startOffset:endOffset])
		i = bodyEnd + 1

		ctes = append(ctes, cte)

		for i < len(tokens) && tokens[i].Kind == tokenizer.KindDocComment {
			i++
		}
		if i < len(tokens) && tokens[i].Kind == tokenizer.KindSymbol && tokens[i].Text == "," {
			i++
			continue
		}
		break
	}

	return ctes, i, diags
}

type variadicGroup struct {
	placeholderIdxs []int
	numberIdxs      []int
	numbers         []int
}

func detectVariadicGroups(tokens []tokenizer.Token) map[int]variadicGroup {
	groups := make(map[int]variadicGroup)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind != tokenizer.KindKeyword && tok.Kind != tokenizer.KindIdentifier {
			continue
		}
		if strings.ToUpper(tok.Text) != "IN" {
			continue
		}

		j := i + 1
		for j < len(tokens) && tokens[j].Kind == tokenizer.KindDocComment {
			j++
		}
		if j >= len(tokens) || tokens[j].Kind != tokenizer.KindSymbol || tokens[j].Text != "(" {
			continue
		}

		depth := 1
		placeholderIdxs := make([]int, 0, 4)
		numberIdxs := make([]int, 0, 4)
		numbers := make([]int, 0, 4)
		valid := true
		hasNumber := false
		hasPlain := false

		for k := j + 1; k < len(tokens) && depth > 0; {
			t := tokens[k]
			if t.Kind == tokenizer.KindDocComment {
				k++
				continue
			}
			if t.Kind == tokenizer.KindSymbol {
				switch t.Text {
				case "(":
					depth++
					k++
					continue
				case ")":
					depth--
					k++
					continue
				case ",":
					k++
					continue
				case "?":
					if depth != 1 {
						valid = false
						break
					}
					placeholderIdxs = append(placeholderIdxs, k)
					numVal := 0
					numIdx := -1
					if k+1 < len(tokens) {
						next := tokens[k+1]
						if next.Kind == tokenizer.KindNumber && next.Line == t.Line && next.Column == t.Column+1 {
							parsed, err := strconv.Atoi(next.Text)
							if err != nil || parsed <= 0 {
								valid = false
								break
							}
							numVal = parsed
							numIdx = k + 1
							numberIdxs = append(numberIdxs, numIdx)
							hasNumber = true
							k += 2
							numbers = append(numbers, numVal)
							continue
						}
					}
					hasPlain = true
					numbers = append(numbers, numVal)
					k++
					continue
				default:
					if depth == 1 {
						valid = false
						break
					}
					k++
					continue
				}
			} else if t.Kind == tokenizer.KindEOF {
				valid = false
				break
			} else {
				if depth == 1 {
					valid = false
					break
				}
				k++
				continue
			}
		}

		if !valid || depth != 0 {
			continue
		}
		if len(placeholderIdxs) < 2 {
			continue
		}
		if hasNumber && hasPlain {
			continue
		}
		if hasNumber {
			for idx := 1; idx < len(numbers); idx++ {
				if numbers[idx] != numbers[0]+idx {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}
		}

		groups[placeholderIdxs[0]] = variadicGroup{
			placeholderIdxs: placeholderIdxs,
			numberIdxs:      numberIdxs,
			numbers:         numbers,
		}
	}
	return groups
}

func nextAvailableOrder(used map[int]int) int {
	order := 1
	for {
		if _, exists := used[order]; !exists {
			return order
		}
		order++
	}
}

func collectParams(tokens []tokenizer.Token, blk block.Block) ([]Param, []Diagnostic) {
	params := make([]Param, 0, 4)
	diags := make([]Diagnostic, 0, 2)
	numbered := make(map[int]int)
	named := make(map[string]int)
	groups := detectVariadicGroups(tokens)
	skipIndices := make(map[int]struct{})

	i := 0
	for i < len(tokens) {
		if _, skip := skipIndices[i]; skip {
			i++
			continue
		}
		tok := tokens[i]
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "?":
				if group, ok := groups[i]; ok {
					conflict := false
					for _, num := range group.numbers {
						if num <= 0 {
							continue
						}
						if _, exists := numbered[num]; exists {
							conflict = true
							break
						}
					}
					if !conflict {
						actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
						order := 0
						if len(group.numbers) > 0 && group.numbers[0] > 0 {
							order = group.numbers[0]
						} else {
							order = nextAvailableOrder(numbered)
						}
						name := fmt.Sprintf("arg%d", order)
						params = append(params, Param{
							Name:          name,
							Style:         ParamStylePositional,
							Order:         order,
							Line:          actualLine,
							Column:        actualColumn,
							IsVariadic:    true,
							VariadicCount: len(group.placeholderIdxs),
						})
						paramIdx := len(params) - 1
						numbered[order] = paramIdx
						for _, num := range group.numbers {
							if num <= 0 {
								continue
							}
							numbered[num] = paramIdx
						}
						for _, idx := range group.placeholderIdxs[1:] {
							skipIndices[idx] = struct{}{}
						}
						for _, idx := range group.numberIdxs {
							skipIndices[idx] = struct{}{}
						}
						i++
						continue
					}
					delete(groups, i)
				}

				order := nextAvailableOrder(numbered)
				actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
				if i+1 < len(tokens) {
					nextTok := tokens[i+1]
					if nextTok.Kind == tokenizer.KindNumber && nextTok.Line == tok.Line && nextTok.Column == tok.Column+1 {
						parsed, err := strconv.Atoi(nextTok.Text)
						if err != nil || parsed <= 0 {
							diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "invalid positional parameter index %s", nextTok.Text))
						} else {
							order = parsed
							if idx, exists := numbered[order]; exists {
								if params[idx].Name != fmt.Sprintf("arg%d", order) {
									diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "duplicate positional parameter %d with conflicting name", order))
								}
							} else {
								params = append(params, Param{
									Name:   fmt.Sprintf("arg%d", order),
									Style:  ParamStylePositional,
									Order:  order,
									Line:   actualLine,
									Column: actualColumn,
								})
								numbered[order] = len(params) - 1
							}
						}
						i += 2
						continue
					}
				}
				params = append(params, Param{
					Name:   fmt.Sprintf("arg%d", order),
					Style:  ParamStylePositional,
					Order:  order,
					Line:   actualLine,
					Column: actualColumn,
				})
				numbered[order] = len(params) - 1
				i++
				continue
			case ":":
				if i+1 < len(tokens) {
					nameTok := tokens[i+1]
					if nameTok.Kind == tokenizer.KindIdentifier {
						raw := tokenizer.NormalizeIdentifier(nameTok.Text)
						key := strings.ToLower(raw)
						camel := camelCaseParam(raw)
						actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
						if idx, exists := named[key]; exists {
							if params[idx].Name != camel {
								diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "parameter %s resolves to conflicting name %q", raw, camel))
							}
						} else {
							params = append(params, Param{
								Name:   camel,
								Style:  ParamStyleNamed,
								Order:  len(params) + 1,
								Line:   actualLine,
								Column: actualColumn,
							})
							named[key] = len(params) - 1
						}
						i += 2
						continue
					}
				}
			}
		}
		i++
	}

	return params, diags
}

type positionIndex struct {
	sql         string
	lineOffsets []int
}

func newPositionIndex(sql string) positionIndex {
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
	return positionIndex{sql: sql, lineOffsets: offsets}
}

func (p positionIndex) offset(tok tokenizer.Token) int {
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
	return idx
}

func parseSelectColumns(tokens []tokenizer.Token, selectIdx int, blk block.Block, pos positionIndex) ([]Column, []Diagnostic) {
	columns := make([]Column, 0, 4)
	diags := make([]Diagnostic, 0, 2)
	depth := 0
	start := selectIdx + 1
	i := start

	for i < len(tokens) {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindEOF {
			break
		}
		if depth == 0 && tok.Kind == tokenizer.KindKeyword && tok.Text == "FROM" {
			break
		}
		if depth == 0 && tok.Kind == tokenizer.KindKeyword && len(columns) == 0 && (tok.Text == "DISTINCT" || tok.Text == "ALL") {
			start = i + 1
			i++
			continue
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
					exprTokens := trimTokens(tokens[start:i])
					if len(exprTokens) > 0 {
						col, cDiags := buildColumn(exprTokens, blk, pos)
						columns = append(columns, col)
						diags = append(diags, cDiags...)
					}
					start = i + 1
				}
			}
		}
		i++
	}
	exprTokens := trimTokens(tokens[start:i])
	if len(exprTokens) > 0 {
		col, cDiags := buildColumn(exprTokens, blk, pos)
		columns = append(columns, col)
		diags = append(diags, cDiags...)
	}

	return columns, diags
}

func trimTokens(tokens []tokenizer.Token) []tokenizer.Token {
	start := 0
	for start < len(tokens) && tokens[start].Kind == tokenizer.KindSymbol && tokens[start].Text == "," {
		start++
	}
	end := len(tokens)
	for end > start && tokens[end-1].Kind == tokenizer.KindSymbol && tokens[end-1].Text == "," {
		end--
	}
	return tokens[start:end]
}

func buildColumn(tokens []tokenizer.Token, blk block.Block, pos positionIndex) (Column, []Diagnostic) {
	var diags []Diagnostic
	exprTokens, aliasTok, alias, hasAlias := extractAlias(tokens)
	table, columnName, simple := analyzeSimpleColumn(exprTokens)
	if !hasAlias && simple {
		alias = columnName
	}
	if alias == "" && !simple {
		if len(exprTokens) > 0 {
			tok := exprTokens[0]
			diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityError, "result column requires alias"))
		}
	}
	if len(exprTokens) == 0 {
		exprTokens = tokens
	}
	if len(exprTokens) == 0 {
		return Column{}, append(diags, makeDiag(blk, 0, 0, SeverityError, "empty result expression"))
	}

	startTok := exprTokens[0]
	endTok := exprTokens[len(exprTokens)-1]
	startOffset := pos.offset(startTok)
	endOffset := pos.offset(endTok) + len(endTok.Text)
	if endOffset > len(pos.sql) {
		endOffset = len(pos.sql)
	}
	expr := strings.TrimSpace(pos.sql[startOffset:endOffset])
	line, column := actualPosition(blk, startTok.Line, startTok.Column)
	col := Column{
		Expr:        expr,
		Alias:       alias,
		Table:       table,
		Line:        line,
		Column:      column,
		StartOffset: startOffset,
		EndOffset:   endOffset,
	}
	if aliasTok != nil {
		line, column = actualPosition(blk, aliasTok.Line, aliasTok.Column)
		col.Line = line
		col.Column = column
	}
	return col, diags
}

func extractAlias(tokens []tokenizer.Token) ([]tokenizer.Token, *tokenizer.Token, string, bool) {
	if len(tokens) == 0 {
		return tokens, nil, "", false
	}
	depth := 0
	for i := len(tokens) - 1; i >= 0; i-- {
		tok := tokens[i]
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case ")":
				depth++
				continue
			case "(":
				if depth > 0 {
					depth--
				}
				continue
			}
		}
		if depth > 0 {
			continue
		}
		if tok.Kind == tokenizer.KindIdentifier {
			aliasTok := tok
			alias := tokenizer.NormalizeIdentifier(tok.Text)
			if i > 0 && tokens[i-1].Kind == tokenizer.KindKeyword && tokens[i-1].Text == "AS" {
				return tokens[:i-1], &aliasTok, alias, true
			}
			if i == 0 {
				break
			}
			if tokens[i-1].Kind == tokenizer.KindSymbol && tokens[i-1].Text == "." {
				break
			}
			return tokens[:i], &aliasTok, alias, true
		}
		if tok.Kind == tokenizer.KindKeyword && tok.Text == "AS" {
			continue
		}
	}
	return tokens, nil, "", false
}

func analyzeSimpleColumn(tokens []tokenizer.Token) (table string, column string, ok bool) {
	if len(tokens) == 1 && tokens[0].Kind == tokenizer.KindIdentifier {
		column = tokenizer.NormalizeIdentifier(tokens[0].Text)
		return "", column, true
	}
	if len(tokens) == 3 && tokens[0].Kind == tokenizer.KindIdentifier && tokens[1].Kind == tokenizer.KindSymbol && tokens[1].Text == "." && tokens[2].Kind == tokenizer.KindIdentifier {
		table = tokenizer.NormalizeIdentifier(tokens[0].Text)
		column = tokenizer.NormalizeIdentifier(tokens[2].Text)
		return table, column, true
	}
	return "", "", false
}

func camelCaseParam(name string) string {
	if name == "" {
		return "param"
	}
	parts := make([]string, 0, 4)
	var segment strings.Builder
	for _, r := range name {
		switch r {
		case '_', '-', '.':
			if segment.Len() > 0 {
				parts = append(parts, segment.String())
				segment.Reset()
			}
		default:
			segment.WriteRune(r)
		}
	}
	if segment.Len() > 0 {
		parts = append(parts, segment.String())
	}
	if len(parts) == 0 {
		parts = append(parts, name)
	}
	var out strings.Builder
	for i, part := range parts {
		lower := strings.ToLower(part)
		if lower == "" {
			continue
		}
		if i == 0 {
			out.WriteString(lower)
			continue
		}
		first, size := utf8.DecodeRuneInString(lower)
		if first == utf8.RuneError {
			out.WriteString(lower)
			continue
		}
		out.WriteRune(unicode.ToUpper(first))
		out.WriteString(lower[size:])
	}
	ident := out.String()
	if ident == "" {
		ident = "param"
	}
	first, _ := utf8.DecodeRuneInString(ident)
	if !unicode.IsLetter(first) && first != '_' {
		ident = "p" + ident
	}
	if token.Lookup(ident).IsKeyword() {
		ident += "_"
	}
	return ident
}

func makeDiag(blk block.Block, relLine, relColumn int, severity Severity, format string, args ...any) Diagnostic {
	line, column := actualPosition(blk, relLine, relColumn)
	return Diagnostic{
		Path:     blk.Path,
		Line:     line,
		Column:   column,
		Message:  fmt.Sprintf(format, args...),
		Severity: severity,
	}
}

func actualPosition(blk block.Block, relLine, relColumn int) (int, int) {
	line := blk.Line + relLine
	if relLine == 0 {
		line = blk.Line
	}
	return line, relColumn
}
