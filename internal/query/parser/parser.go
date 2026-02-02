// Package parser implements a SQL parser for query validation and parameter extraction.
package parser

import (
	"errors"
	"fmt"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Query represents a parsed SQL query with metadata.
type Query struct {
	Block       block.Block
	Verb        Verb
	Columns     []Column
	Params      []Param
	CTEs        []CTE
	Diagnostics []Diagnostic
}

// Verb indicates the type of SQL statement (SELECT, INSERT, etc.).
type Verb int

const (
	// VerbUnknown indicates an unrecognized statement type.
	VerbUnknown Verb = iota
	// VerbSelect indicates a SELECT statement.
	VerbSelect
	// VerbInsert indicates an INSERT statement.
	VerbInsert
	// VerbUpdate indicates an UPDATE statement.
	VerbUpdate
	// VerbDelete indicates a DELETE statement.
	VerbDelete
)

// Column represents a result column in a SELECT query.
type Column struct {
	Expr        string
	Alias       string
	Table       string
	Line        int
	Column      int
	StartOffset int
	EndOffset   int
}

// CTE represents a Common Table Expression.
type CTE struct {
	Name      string
	Columns   []string
	SelectSQL string
	Line      int
	Column    int
}

// Param represents a query parameter.
type Param struct {
	Name          string
	Style         ParamStyle
	Order         int
	Line          int
	Column        int
	IsVariadic    bool
	VariadicCount int
	StartOffset   int
	EndOffset     int
}

// ParamStyle indicates how parameters are specified (positional or named).
type ParamStyle int

const (
	// ParamStyleUnknown indicates an unrecognized parameter style.
	ParamStyleUnknown ParamStyle = iota
	// ParamStylePositional indicates parameters using '?' or '?NNN'.
	ParamStylePositional
	// ParamStyleNamed indicates parameters using ':name'.
	ParamStyleNamed
)

// Severity indicates the seriousness of a diagnostic.
type Severity int

const (
	// SeverityError indicates a fatal parsing error.
	SeverityError Severity = iota
	// SeverityWarning indicates a non-fatal issue.
	SeverityWarning
)

// Diagnostic represents an issue found during parsing.
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// Parse parses a query block into a structured Query object.
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

	params, paramDiags := collectParams(tokens, blk, posIdx)
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

	var tokErr *tokenizer.Error
	if errors.As(err, &tokErr) {
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
		endOffset = min(endOffset, len(pos.sql))
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
	name            string // for sqlc.slice('name')
}

func isSQLCMacro(tokens []tokenizer.Token, i int) (macroType string, arg string, end int, ok bool) {
	if i+5 >= len(tokens) {
		return "", "", 0, false
	}
	// sqlc . macro ( 'arg' )
	if tokens[i].Kind != tokenizer.KindIdentifier || tokens[i].Text != "sqlc" {
		return "", "", 0, false
	}
	if tokens[i+1].Kind != tokenizer.KindSymbol || tokens[i+1].Text != "." {
		return "", "", 0, false
	}
	if tokens[i+2].Kind != tokenizer.KindIdentifier {
		return "", "", 0, false
	}
	macro := tokens[i+2].Text
	if tokens[i+3].Kind != tokenizer.KindSymbol || tokens[i+3].Text != "(" {
		return "", "", 0, false
	}
	if tokens[i+4].Kind != tokenizer.KindString {
		return "", "", 0, false
	}
	val := tokens[i+4].Text
	// Strip quotes
	if len(val) >= 2 && (val[0] == '\'' || val[0] == '"') {
		val = val[1 : len(val)-1]
	}
	if tokens[i+5].Kind != tokenizer.KindSymbol || tokens[i+5].Text != ")" {
		return "", "", 0, false
	}
	return macro, val, i + 6, true
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

		// Check for sqlc.slice('name') immediately inside IN ( ... )
		// Skip whitespace/comments inside parens if necessary, but sqlc macros usually don't have comments inside
		k := j + 1
		for k < len(tokens) && tokens[k].Kind == tokenizer.KindDocComment {
			k++
		}

		if macro, name, end, ok := isSQLCMacro(tokens, k); ok && macro == "slice" {
			// Check if closed by )
			if end < len(tokens) && tokens[end].Kind == tokenizer.KindSymbol && tokens[end].Text == ")" {
				groups[k] = variadicGroup{
					placeholderIdxs: []int{k}, // Use start of macro as placeholder
					name:            name,
				}
				i = end // Advance main loop
				continue
			}
		}

		depth := 1
		placeholderIdxs := make([]int, 0, 4)
		numberIdxs := make([]int, 0, 4)
		numbers := make([]int, 0, 4)
		valid := true
		hasNumber := false
		hasPlain := false

	TokenLoop:
		for k := j + 1; k < len(tokens) && depth > 0; {
			t := tokens[k]
			if t.Kind == tokenizer.KindDocComment {
				k++
				continue
			}
			switch t.Kind {
			case tokenizer.KindSymbol:
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
						break TokenLoop
					}
					placeholderIdxs = append(placeholderIdxs, k)
					numVal := 0
					if k+1 < len(tokens) {
						next := tokens[k+1]
						if next.Kind == tokenizer.KindNumber && next.Line == t.Line && next.Column == t.Column+1 {
							parsed, err := strconv.Atoi(next.Text)
							if err != nil || parsed <= 0 {
								valid = false
								break TokenLoop
							}
							numVal = parsed
							numIdx := k + 1
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
						break TokenLoop
					}
					k++
					continue
				}
			case tokenizer.KindEOF:
				valid = false
				break TokenLoop
			default:
				if depth == 1 {
					valid = false
					break TokenLoop
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

func collectParams(tokens []tokenizer.Token, blk block.Block, pos positionIndex) ([]Param, []Diagnostic) {
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

		// Check for sqlc.slice from groups
		if group, ok := groups[i]; ok && group.name != "" {
			actualLine, actualColumn := actualPosition(blk, tokens[i].Line, tokens[i].Column)
			startOffset := pos.offset(tokens[i])
			// Macro length: sqlc(4) + .(1) + slice(5) + ((1) + 'name'(len) + )(1)
			// tokens[i] is sqlc. tokens[i+5] is ).
			endTok := tokens[i+5]
			endOffset := pos.offset(endTok) + len(endTok.Text)

			camel := camelCaseParam(group.name)
			params = append(params, Param{
				Name:          camel,
				Style:         ParamStyleNamed,
				Order:         len(params) + 1,
				Line:          actualLine,
				Column:        actualColumn,
				IsVariadic:    true,
				VariadicCount: 0, // 0 indicates dynamic expansion
				StartOffset:   startOffset,
				EndOffset:     endOffset,
			})
			named[strings.ToLower(group.name)] = len(params) - 1
			// Skip the whole macro: sqlc . slice ( 'name' ) = 6 tokens
			for k := range 6 {
				skipIndices[i+k] = struct{}{}
			}
			i++
			continue
		}

		// Check for sqlc.arg / sqlc.narg
		if macro, name, end, ok := isSQLCMacro(tokens, i); ok && (macro == "arg" || macro == "narg") {
			actualLine, actualColumn := actualPosition(blk, tokens[i].Line, tokens[i].Column)
			startOffset := pos.offset(tokens[i])
			endTok := tokens[end-1]
			endOffset := pos.offset(endTok) + len(endTok.Text)

			camel := camelCaseParam(name)
			key := strings.ToLower(name)

			if idx, exists := named[key]; exists {
				if params[idx].Name != camel {
					diags = append(diags, makeDiag(blk, tokens[i].Line, tokens[i].Column, SeverityError, "parameter %s resolves to conflicting name %q", name, camel))
				}
			} else {
				params = append(params, Param{
					Name:        camel,
					Style:       ParamStyleNamed,
					Order:       len(params) + 1,
					Line:        actualLine,
					Column:      actualColumn,
					StartOffset: startOffset,
					EndOffset:   endOffset,
				})
				named[key] = len(params) - 1
			}
			// Skip macro tokens
			for k := i; k < end; k++ {
				skipIndices[k] = struct{}{}
			}
			i = end
			continue
		}

		tok := tokens[i]

		// Handle PostgreSQL-style positional parameters ($1, $2, etc.)
		if tok.Kind == tokenizer.KindParam {
			newParams, newDiags, consumed := handlePostgresParam(tokens, i, blk, pos, numbered, params)
			params = newParams
			diags = append(diags, newDiags...)
			i += consumed
			if consumed > 0 {
				continue
			}
		}

		if tok.Kind != tokenizer.KindSymbol {
			i++
			continue
		}

		switch tok.Text {
		case "?":
			newParams, newDiags, consumed := handlePositionalParam(tokens, i, blk, pos, groups, numbered, params, skipIndices)
			params = newParams
			diags = append(diags, newDiags...)
			i += consumed
			if consumed > 0 {
				continue
			}
		case ":":
			newParams, newDiags, consumed := handleNamedParam(tokens, i, blk, pos, named, params)
			params = newParams
			diags = append(diags, newDiags...)
			i += consumed
			if consumed > 0 {
				continue
			}
		}
		i++
	}

	return params, diags
}

// handlePositionalParam processes a "?" positional parameter.
// Returns the updated params slice, diagnostics, and the number of tokens consumed.
func handlePositionalParam(
	tokens []tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	groups map[int]variadicGroup,
	numbered map[int]int,
	params []Param,
	skipIndices map[int]struct{},
) ([]Param, []Diagnostic, int) {
	tok := tokens[idx]

	// Check for variadic group first
	if group, ok := groups[idx]; ok {
		result, consumed := tryProcessVariadicGroup(tokens, tok, idx, blk, pos, group, numbered, params, skipIndices)
		if consumed > 0 {
			return result, nil, consumed
		}
		delete(groups, idx)
	}

	// Check for numbered parameter (e.g., ?1)
	if result, diags, consumed := tryProcessNumberedParam(tokens, idx, blk, pos, numbered, params); consumed > 0 {
		return result, diags, consumed
	}

	// Simple positional parameter
	return processSimplePositionalParam(tokens, tok, idx, blk, pos, numbered, params)
}

// tryProcessVariadicGroup handles variadic parameter groups like ?1,?2,?3.
func tryProcessVariadicGroup(
	tokens []tokenizer.Token,
	tok tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	group variadicGroup,
	numbered map[int]int,
	params []Param,
	skipIndices map[int]struct{},
) ([]Param, int) {
	// Check for conflicts
	for _, num := range group.numbers {
		if num <= 0 {
			continue
		}
		if _, exists := numbered[num]; exists {
			return nil, 0
		}
	}

	actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
	startOffset := pos.offset(tok)
	endTok := tokens[group.placeholderIdxs[len(group.placeholderIdxs)-1]]
	endOffset := pos.offset(endTok) + len(endTok.Text)

	order := nextAvailableOrder(numbered)
	if len(group.numbers) > 0 && group.numbers[0] > 0 {
		order = group.numbers[0]
	}

	paramName := inferParamName(tokens, idx, numbered)
	if paramName == "" {
		paramName = fmt.Sprintf("arg%d", order)
	}

	params = append(params, Param{
		Name:          paramName,
		Style:         ParamStylePositional,
		Order:         order,
		Line:          actualLine,
		Column:        actualColumn,
		IsVariadic:    true,
		VariadicCount: len(group.placeholderIdxs),
		StartOffset:   startOffset,
		EndOffset:     endOffset,
	})

	paramIdx := len(params) - 1
	numbered[order] = paramIdx
	for _, num := range group.numbers {
		if num > 0 {
			numbered[num] = paramIdx
		}
	}
	for _, i := range group.placeholderIdxs[1:] {
		skipIndices[i] = struct{}{}
	}
	for _, i := range group.numberIdxs {
		skipIndices[i] = struct{}{}
	}

	return params, 1
}

// tryProcessNumberedParam handles numbered positional parameters like ?1.
func tryProcessNumberedParam(
	tokens []tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	numbered map[int]int,
	params []Param,
) ([]Param, []Diagnostic, int) {
	tok := tokens[idx]
	if idx+1 >= len(tokens) {
		return nil, nil, 0
	}

	nextTok := tokens[idx+1]
	if nextTok.Kind != tokenizer.KindNumber || nextTok.Line != tok.Line || nextTok.Column != tok.Column+1 {
		return nil, nil, 0
	}

	parsed, err := strconv.Atoi(nextTok.Text)
	if err != nil || parsed <= 0 {
		return nil, []Diagnostic{makeDiag(blk, tok.Line, tok.Column, SeverityError, "invalid positional parameter index %s", nextTok.Text)}, 2
	}

	actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
	startOffset := pos.offset(tok)
	endOffset := pos.offset(nextTok) + len(nextTok.Text)

	paramName := inferParamName(tokens, idx, numbered)
	if paramName == "" {
		paramName = fmt.Sprintf("arg%d", parsed)
	}

	if existingIdx, exists := numbered[parsed]; exists {
		if params[existingIdx].Name != paramName {
			return nil, []Diagnostic{makeDiag(blk, tok.Line, tok.Column, SeverityError, "duplicate positional parameter %d with conflicting name", parsed)}, 2
		}
		return params, nil, 2
	}

	params = append(params, Param{
		Name:        paramName,
		Style:       ParamStylePositional,
		Order:       parsed,
		Line:        actualLine,
		Column:      actualColumn,
		StartOffset: startOffset,
		EndOffset:   endOffset,
	})
	numbered[parsed] = len(params) - 1
	return params, nil, 2
}

// processSimplePositionalParam handles a simple "?" parameter.
func processSimplePositionalParam(
	tokens []tokenizer.Token,
	tok tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	numbered map[int]int,
	params []Param,
) ([]Param, []Diagnostic, int) {
	order := nextAvailableOrder(numbered)
	actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
	startOffset := pos.offset(tok)
	endOffset := startOffset + len(tok.Text)

	paramName := inferParamName(tokens, idx, numbered)
	if paramName == "" {
		paramName = fmt.Sprintf("arg%d", order)
	}

	params = append(params, Param{
		Name:        paramName,
		Style:       ParamStylePositional,
		Order:       order,
		Line:        actualLine,
		Column:      actualColumn,
		StartOffset: startOffset,
		EndOffset:   endOffset,
	})
	numbered[order] = len(params) - 1
	return params, nil, 1
}

// handlePostgresParam processes a PostgreSQL-style positional parameter ($1, $2, etc.)
// Returns the updated params slice, diagnostics, and the number of tokens consumed.
func handlePostgresParam(
	tokens []tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	numbered map[int]int,
	params []Param,
) ([]Param, []Diagnostic, int) {
	tok := tokens[idx]

	// Extract the parameter number from $N
	paramText := tok.Text
	if len(paramText) < 2 || paramText[0] != '$' {
		return params, nil, 0
	}

	paramNum, err := strconv.Atoi(paramText[1:])
	if err != nil || paramNum <= 0 {
		return params, []Diagnostic{{
			Line:    tok.Line,
			Column:  tok.Column,
			Message: fmt.Sprintf("invalid PostgreSQL parameter: %s", paramText),
		}}, 1
	}

	actualLine, actualColumn := actualPosition(blk, tok.Line, tok.Column)
	startOffset := pos.offset(tok)
	endOffset := startOffset + len(tok.Text)

	paramName := inferParamName(tokens, idx, numbered)
	if paramName == "" {
		paramName = fmt.Sprintf("arg%d", paramNum)
	}

	params = append(params, Param{
		Name:        paramName,
		Style:       ParamStylePositional,
		Order:       paramNum,
		Line:        actualLine,
		Column:      actualColumn,
		StartOffset: startOffset,
		EndOffset:   endOffset,
	})
	numbered[paramNum] = len(params) - 1
	return params, nil, 1
}

// handleNamedParam processes a ":" named parameter.
// Returns the updated params slice, diagnostics, and the number of tokens consumed.
func handleNamedParam(
	tokens []tokenizer.Token,
	idx int,
	blk block.Block,
	pos positionIndex,
	named map[string]int,
	params []Param,
) ([]Param, []Diagnostic, int) {
	if idx+1 >= len(tokens) {
		return params, nil, 0
	}

	nameTok := tokens[idx+1]
	if nameTok.Kind != tokenizer.KindIdentifier {
		return params, nil, 0
	}

	raw := tokenizer.NormalizeIdentifier(nameTok.Text)
	key := strings.ToLower(raw)
	camel := camelCaseParam(raw)

	actualLine, actualColumn := actualPosition(blk, tokens[idx].Line, tokens[idx].Column)
	startOffset := pos.offset(tokens[idx])
	endOffset := pos.offset(nameTok) + len(nameTok.Text)

	if existingIdx, exists := named[key]; exists {
		if params[existingIdx].Name != camel {
			return params, []Diagnostic{
				makeDiag(blk, tokens[idx].Line, tokens[idx].Column, SeverityError, "parameter %s resolves to conflicting name %q", raw, camel),
			}, 2
		}
		return params, nil, 2
	}

	params = append(params, Param{
		Name:        camel,
		Style:       ParamStyleNamed,
		Order:       len(params) + 1,
		Line:        actualLine,
		Column:      actualColumn,
		StartOffset: startOffset,
		EndOffset:   endOffset,
	})
	named[key] = len(params) - 1
	return params, nil, 2
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
			diags = append(diags, makeDiag(blk, tok.Line, tok.Column, SeverityWarning, "result column requires alias"))
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
	endOffset = min(endOffset, len(pos.sql))
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
	if len(tokens) == 1 {
		if tokens[0].Kind == tokenizer.KindIdentifier {
			column = tokenizer.NormalizeIdentifier(tokens[0].Text)
			return "", column, true
		}
		if tokens[0].Kind == tokenizer.KindSymbol && tokens[0].Text == "*" {
			return "", "*", true
		}
	}
	if len(tokens) == 3 && tokens[1].Kind == tokenizer.KindSymbol && tokens[1].Text == "." {
		if tokens[0].Kind == tokenizer.KindIdentifier &&
			(tokens[2].Kind == tokenizer.KindIdentifier || (tokens[2].Kind == tokenizer.KindSymbol && tokens[2].Text == "*")) {
			table = tokenizer.NormalizeIdentifier(tokens[0].Text)
			column = tokenizer.NormalizeIdentifier(tokens[2].Text)
			return table, column, true
		}
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

// inferParamName attempts to infer a meaningful parameter name from the surrounding context.
// It looks backward from the parameter position for comparison operators and column names.
// Returns empty string if no meaningful name can be inferred or if inference would be ambiguous.
func inferParamName(tokens []tokenizer.Token, paramIdx int, usedOrders map[int]int) string {
	if !isValidParamToken(tokens, paramIdx) {
		return ""
	}

	order, paramOrder, orderCount := countParameterOrders(tokens)

	// Check for ambiguous parameter usage
	if shouldSkipInference(paramIdx, paramOrder, orderCount, order, usedOrders) {
		return ""
	}

	// Look backward for column name
	return findColumnNameForParam(tokens, paramIdx)
}

// isValidParamToken checks if the token at paramIdx is a valid parameter placeholder.
func isValidParamToken(tokens []tokenizer.Token, paramIdx int) bool {
	if paramIdx <= 0 || paramIdx >= len(tokens) {
		return false
	}
	paramTok := tokens[paramIdx]
	return paramTok.Kind == tokenizer.KindSymbol && paramTok.Text == "?"
}

// countParameterOrders counts how many times the first encountered order appears.
func countParameterOrders(tokens []tokenizer.Token) (order, paramOrder, orderCount int) {
	for i, tok := range tokens {
		if tok.Kind != tokenizer.KindSymbol || tok.Text != "?" {
			continue
		}
		order, paramOrder, orderCount = updateOrderCount(tokens, i, order, paramOrder, orderCount)
	}
	return order, paramOrder, orderCount
}

// updateOrderCount updates the order tracking based on a single token.
func updateOrderCount(tokens []tokenizer.Token, idx, order, paramOrder, orderCount int) (int, int, int) {
	if idx+1 >= len(tokens) || tokens[idx+1].Kind != tokenizer.KindNumber {
		// Plain "?" without a number - only set paramOrder for the first one
		if paramOrder == 0 {
			return order, idx, orderCount
		}
		return order, paramOrder, orderCount
	}

	parsed, _ := strconv.Atoi(tokens[idx+1].Text)
	if parsed <= 0 {
		return order, paramOrder, orderCount
	}

	if order == 0 {
		order = parsed
		paramOrder = idx
	}
	if parsed == order {
		orderCount++
	}
	return order, paramOrder, orderCount
}

// shouldSkipInference determines if parameter name inference should be skipped.
func shouldSkipInference(paramIdx, paramOrder, orderCount, order int, usedOrders map[int]int) bool {
	if orderCount > 1 {
		return true
	}
	if paramIdx != paramOrder {
		return true
	}
	if usedOrders[order] > 0 && order > 0 {
		return true
	}
	return false
}

// findColumnNameForParam searches backward from paramIdx for a column name.
func findColumnNameForParam(tokens []tokenizer.Token, paramIdx int) string {
	for j := paramIdx - 1; j >= 0; j-- {
		tok := tokens[j]

		if tok.Kind == tokenizer.KindEOF {
			break
		}

		name := tryExtractColumnName(tokens, j)
		if name == "_stop_" {
			return ""
		}
		if name != "" {
			return name
		}

		if shouldStopSearch(tok) {
			break
		}
	}
	return ""
}

// tryExtractColumnName attempts to extract a column name when an operator is found.
// Returns the column name, "_stop_" to indicate search should stop, or "" to continue.
func tryExtractColumnName(tokens []tokenizer.Token, opIdx int) string {
	tok := tokens[opIdx]

	// Handle comparison operators (symbols)
	if tok.Kind == tokenizer.KindSymbol {
		switch tok.Text {
		case "=", "<", ">", "<=", ">=", "!=", "<>":
			return findColumnBeforeOperator(tokens, opIdx)
		case ")", ",", "+", "-", "*", "/", "%":
			return "_stop_"
		case "(":
			// Don't stop on '(', just continue searching
			return ""
		}
	}

	// Handle keyword operators (case-insensitive)
	if tok.Kind == tokenizer.KindKeyword {
		upper := strings.ToUpper(tok.Text)
		switch upper {
		case "LIKE", "IN", "BETWEEN":
			return findColumnBeforeOperator(tokens, opIdx)
		}
	}

	// Handle operators that might be tokenized as identifiers (not keywords in SQLite)
	// e.g., LIKE, IN, BETWEEN are not in the keywords list
	if tok.Kind == tokenizer.KindIdentifier {
		upper := strings.ToUpper(tok.Text)
		switch upper {
		case "LIKE", "IN", "BETWEEN":
			return findColumnBeforeOperator(tokens, opIdx)
		}
	}

	return ""
}

// findColumnBeforeOperator searches for an identifier before the operator.
func findColumnBeforeOperator(tokens []tokenizer.Token, opIdx int) string {
	for k := opIdx - 1; k >= 0; k-- {
		prevTok := tokens[k]
		if prevTok.Kind == tokenizer.KindIdentifier {
			return camelCaseParam(prevTok.Text)
		}
		if prevTok.Kind == tokenizer.KindSymbol && prevTok.Text != "(" && prevTok.Text != "," {
			break
		}
	}
	return ""
}

// shouldStopSearch checks if the search should stop at this token.
func shouldStopSearch(tok tokenizer.Token) bool {
	// Check both KindKeyword and KindIdentifier since many SQL keywords
	// are not in the tokenizer's keywords list
	if tok.Kind != tokenizer.KindKeyword && tok.Kind != tokenizer.KindIdentifier {
		return false
	}
	upper := strings.ToUpper(tok.Text)
	// Note: SET and ON are not in this list because they are contexts where
	// we want to find column names (SET col = ?, JOIN ... ON col = ?)
	stopKeywords := []string{"WHERE", "AND", "OR", "VALUES", "HAVING", "ORDER", "GROUP", "BY", "LIMIT", "OFFSET", "SELECT", "FROM", "INSERT", "UPDATE", "DELETE"}
	return slices.Contains(stopKeywords, upper)
}
