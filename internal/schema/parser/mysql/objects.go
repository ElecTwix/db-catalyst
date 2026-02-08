package mysql

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// parseCreateIndex handles CREATE INDEX statements.
func (ps *parserState) parseCreateIndex(unique bool) {
	createTok := ps.previous()

	// Check for FULLTEXT or SPATIAL
	isFullText := false
	if ps.matchKeyword("FULLTEXT") {
		isFullText = true
		ps.advance()
	} else if ps.matchKeyword("SPATIAL") {
		ps.advance()
	}

	// Skip INDEX or KEY
	if ps.matchKeyword("INDEX") || ps.matchKeyword("KEY") {
		ps.advance()
	}

	// Parse index name (optional for MySQL)
	indexName := ""
	if ps.current().Kind == tokenizer.KindIdentifier {
		name, _, _ := ps.parseIdentifier(true)
		indexName = name
	}

	// Expect ON
	if !ps.matchKeyword("ON") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected ON in CREATE INDEX")
		ps.sync()
		return
	}
	ps.advance()

	// Parse table name
	tableName, _, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	// Parse column list
	cols, last, ok := ps.parseColumnNameList()
	if !ok {
		return
	}

	tail := last

	idx := &model.Index{
		Name:    indexName,
		Unique:  unique,
		Columns: cols,
		Span:    tokenizer.SpanBetween(createTok, tail),
	}
	_ = isFullText

	table := ps.lookupTable(tableName)
	if table == nil {
		ps.addDiagSpan(idx.Span, diagnostic.SeverityError,
			"index %q references unknown table %q", indexName, tableName)
	} else {
		table.Indexes = append(table.Indexes, idx)
	}

	if ps.matchSymbol(";") {
		ps.advance()
	}
}

// parseCreateView handles CREATE VIEW statements.
func (ps *parserState) parseCreateView() {
	createTok := ps.previous()

	// Check for OR REPLACE
	if ps.matchKeyword("OR") {
		ps.advance()
		if ps.matchKeyword("REPLACE") {
			ps.advance()
		}
	}

	// Skip ALGORITHM, DEFINER, SQL SECURITY
	for ps.matchKeyword("ALGORITHM") || ps.matchKeyword("DEFINER") || ps.matchKeyword("SQL") {
		ps.advance()
		if ps.matchSymbol("=") {
			ps.advance()
		}
		// Skip value
		ps.advance()
	}

	// Parse view name
	name, _, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	view := &model.View{
		Name: name,
		Doc:  ps.takeDoc(),
	}

	// Check for column list (optional)
	if ps.matchSymbol("(") {
		ps.skipBalancedParentheses()
	}

	// Expect AS
	if !ps.matchKeyword("AS") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected AS in CREATE VIEW")
		ps.sync()
		return
	}
	ps.advance()

	// Collect SQL tokens
	var sqlTokens []tokenizer.Token
	var last tokenizer.Token = createTok

	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ";" {
			last = tok
			ps.advance()
			break
		}
		sqlTokens = append(sqlTokens, tok)
		last = tok
		ps.advance()
	}

	view.SQL = rebuildSQL(sqlTokens)
	view.Span = tokenizer.SpanBetween(createTok, last)

	key := canonicalName(name)
	if _, exists := ps.catalog.Views[key]; exists {
		ps.addDiagSpan(view.Span, diagnostic.SeverityError, "duplicate view %q", name)
		return
	}

	ps.catalog.Views[key] = view
}

// parseDefaultValue parses a DEFAULT value.
func (ps *parserState) parseDefaultValue() (*model.Value, tokenizer.Token) {
	tok := ps.current()

	switch tok.Kind {
	case tokenizer.KindNumber:
		ps.advance()
		return &model.Value{
			Kind: model.ValueKindNumber,
			Text: tok.Text,
			Span: tokenizer.NewSpan(tok),
		}, tok

	case tokenizer.KindString:
		ps.advance()
		return &model.Value{
			Kind: model.ValueKindString,
			Text: tok.Text,
			Span: tokenizer.NewSpan(tok),
		}, tok

	case tokenizer.KindKeyword:
		upper := tok.Text
		ps.advance()

		// Handle CURRENT_TIMESTAMP with optional ON UPDATE
		if upper == "CURRENT_TIMESTAMP" || upper == "CURRENT_TIMESTAMP()" {
			// Check for (precision)
			if ps.matchSymbol("(") {
				ps.skipBalancedParentheses()
			}
			return &model.Value{
				Kind: model.ValueKindKeyword,
				Text: upper,
				Span: tokenizer.NewSpan(tok),
			}, tok
		}

		return &model.Value{
			Kind: model.ValueKindKeyword,
			Text: tok.Text,
			Span: tokenizer.NewSpan(tok),
		}, tok
	}

	// Handle expressions
	tokens, last := ps.collectDefaultExpressionTokens()
	if len(tokens) == 0 {
		return nil, tok
	}

	return &model.Value{
		Kind: model.ValueKindUnknown,
		Text: rebuildSQL(tokens),
		Span: tokenizer.SpanBetween(tokens[0], last),
	}, last
}

// collectDefaultExpressionTokens collects tokens for a DEFAULT expression.
func (ps *parserState) collectDefaultExpressionTokens() ([]tokenizer.Token, tokenizer.Token) {
	var tokens []tokenizer.Token
	var last tokenizer.Token
	depth := 0

	for !ps.isEOF() {
		tok := ps.current()
		if ps.shouldBreakTokenCollection(tok, len(tokens) == 0, depth) {
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
		}

		tokens = append(tokens, tok)
		last = tok
		ps.advance()
	}

	return tokens, last
}

// shouldBreakTokenCollection determines if token collection should stop.
func (ps *parserState) shouldBreakTokenCollection(tok tokenizer.Token, isFirstToken bool, depth int) bool {
	if !isFirstToken && depth != 0 {
		return false
	}

	if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")" || tok.Text == ";") {
		return true
	}

	if tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
		return true
	}

	return false
}

// rebuildSQL reconstructs SQL from tokens.
func rebuildSQL(tokens []tokenizer.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	var b strings.Builder
	var prev string

	for _, tok := range tokens {
		part := tok.Text
		if b.Len() == 0 {
			b.WriteString(part)
			prev = part
			continue
		}

		if needsSpace(prev, part) {
			b.WriteByte(' ')
		}
		b.WriteString(part)
		prev = part
	}

	return strings.TrimSpace(b.String())
}

// needsSpace determines if a space is needed between tokens.
func needsSpace(prev, next string) bool {
	if prev == "" || next == "" {
		return false
	}

	noSpaceBefore := map[string]struct{}{
		",": {},
		")": {},
		".": {},
	}
	noSpaceAfter := map[string]struct{}{
		"(": {},
		".": {},
	}

	if _, ok := noSpaceBefore[next]; ok {
		return false
	}
	if _, ok := noSpaceAfter[prev]; ok {
		return false
	}

	return true
}
