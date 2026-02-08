package postgres

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// parseTableConstraint parses table-level constraints.
func (ps *parserState) parseTableConstraint(table *model.Table) {
	var constraintName string
	if ps.matchKeyword("CONSTRAINT") {
		ps.advance()
		name, _, ok := ps.parseIdentifier(true)
		if ok {
			constraintName = name
		}
	}

	tok := ps.current()
	if tok.Kind != tokenizer.KindKeyword {
		ps.addDiagToken(tok, diagnostic.SeverityError, "expected PRIMARY, UNIQUE, FOREIGN, CHECK, or EXCLUDE after CONSTRAINT")
		ps.skipUntilClauseEnd()
		return
	}

	switch tok.Text {
	case "PRIMARY":
		start := ps.advance()
		if !ps.matchKeyword("KEY") {
			ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected KEY after PRIMARY")
		} else {
			ps.advance()
		}
		cols, last, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		pk := &model.PrimaryKey{
			Name:    constraintName,
			Columns: cols,
			Span:    tokenizer.SpanBetween(start, last),
		}
		if table.PrimaryKey != nil {
			ps.addDiagSpan(pk.Span, diagnostic.SeverityError, "table %s already has a primary key", table.Name)
			return
		}
		table.PrimaryKey = pk

	case "UNIQUE":
		start := ps.advance()
		cols, last, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		table.UniqueKeys = append(table.UniqueKeys, &model.UniqueKey{
			Name:    constraintName,
			Columns: cols,
			Span:    tokenizer.SpanBetween(start, last),
		})

	case "FOREIGN":
		start := ps.advance()
		if !ps.matchKeyword("KEY") {
			ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected KEY after FOREIGN")
			return
		}
		ps.advance()
		cols, _, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		if !ps.matchKeyword("REFERENCES") {
			ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected REFERENCES in foreign key constraint")
			ps.skipUntilClauseEnd()
			return
		}
		ps.advance()
		ref, refLast, ok := ps.parseForeignKeyRef()
		if !ok {
			return
		}
		fkEnd := refLast
		if extra := ps.skipForeignKeyActions(); extra.Line != 0 {
			fkEnd = extra
		}
		fk := &model.ForeignKey{
			Name:    constraintName,
			Columns: cols,
			Ref:     *ref,
			Span:    tokenizer.SpanBetween(start, fkEnd),
		}
		table.ForeignKeys = append(table.ForeignKeys, fk)

	case "CHECK":
		ps.advance()
		ps.skipCheckConstraint()

	case "EXCLUDE":
		// PostgreSQL-specific EXCLUDE constraint
		ps.advance()
		ps.skipBalancedParentheses()
		ps.skipUntilClauseEnd()

	default:
		ps.addDiagToken(tok, diagnostic.SeverityError, "unsupported table constraint %s", tok.Text)
		ps.skipUntilClauseEnd()
	}
}

// parseColumnNameList parses a list of column names in parentheses.
func (ps *parserState) parseColumnNameList() ([]string, tokenizer.Token, bool) {
	if !ps.expectSymbol("(") {
		return nil, ps.current(), false
	}
	_ = ps.previous() // consume opening paren

	var names []string
	var last tokenizer.Token

	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
			last = ps.advance()
			return names, last, true
		}

		name, nameTok, ok := ps.parseIdentifier(true)
		if !ok {
			return names, nameTok, false
		}
		names = append(names, name)
		last = nameTok

		// Skip ordering tokens (ASC, DESC) and collation
		last = ps.skipColumnOrderingTokens(last)

		if ps.matchSymbol(",") {
			last = ps.advance()
			continue
		}
	}

	ps.addDiagToken(ps.current(), diagnostic.SeverityError, "unexpected EOF in column list")
	return names, last, false
}

// skipColumnOrderingTokens skips ASC, DESC, and collation clauses.
func (ps *parserState) skipColumnOrderingTokens(last tokenizer.Token) tokenizer.Token {
	depth := 0
	for !ps.isEOF() {
		tok := ps.current()
		if depth == 0 && tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")") {
			break
		}

		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				} else {
					return last
				}
			}
		}

		// Skip ASC, DESC, COLLATE keywords
		if tok.Kind == tokenizer.KindKeyword {
			switch tok.Text {
			case "ASC", "DESC", "COLLATE", "NULLS", "FIRST", "LAST":
				last = tok
				ps.advance()
				continue
			}
		}

		last = tok
		ps.advance()
	}
	return last
}

// skipBalancedParentheses skips balanced parentheses.
func (ps *parserState) skipBalancedParentheses() tokenizer.Token {
	if !ps.matchSymbol("(") {
		return tokenizer.Token{}
	}
	open := ps.advance()
	depth := 1
	last := open

	for !ps.isEOF() && depth > 0 {
		tok := ps.current()
		last = tok
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				depth--
			}
		}
		ps.advance()
	}

	return last
}

// skipCheckConstraint skips CHECK constraint expression.
func (ps *parserState) skipCheckConstraint() tokenizer.Token {
	var last tokenizer.Token
	if ps.matchSymbol("(") {
		last = ps.skipBalancedParentheses()
	}

	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")") {
			break
		}
		if tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
			break
		}
		last = tok
		ps.advance()
	}

	return last
}

// skipForeignKeyActions skips ON DELETE/UPDATE actions.
func (ps *parserState) skipForeignKeyActions() tokenizer.Token {
	var last tokenizer.Token
	depth := 0

	for !ps.isEOF() {
		tok := ps.current()

		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth == 0 {
					return last
				}
				depth--
			case ",":
				if depth == 0 {
					return last
				}
			case ";":
				if depth == 0 {
					return last
				}
			}
		}

		if depth == 0 && tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
			return last
		}

		last = tok
		ps.advance()
	}

	return last
}

// parseForeignKeyRef parses a foreign key reference (table and optional columns).
func (ps *parserState) parseForeignKeyRef() (*model.ForeignKeyRef, tokenizer.Token, bool) {
	name, nameTok, ok := ps.parseObjectName()
	if !ok {
		return nil, nameTok, false
	}

	ref := &model.ForeignKeyRef{
		Table: name,
		Span:  tokenizer.NewSpan(nameTok),
	}
	last := nameTok

	if ps.matchSymbol("(") {
		cols, closeTok, ok := ps.parseColumnNameList()
		if !ok {
			return nil, closeTok, false
		}
		ref.Columns = cols
		last = closeTok
	}

	ref.Span = tokenizer.SpanBetween(nameTok, last)
	return ref, last, true
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
		// Handle NULL, TRUE, FALSE, CURRENT_TIMESTAMP, etc.
		ps.advance()
		return &model.Value{
			Kind: model.ValueKindKeyword,
			Text: tok.Text,
			Span: tokenizer.NewSpan(tok),
		}, tok
	}

	// Handle expressions and function calls
	tokens, last := ps.collectDefaultExpressionTokens()
	if len(tokens) == 0 {
		return nil, tok
	}

	kind := model.ValueKindUnknown
	allKeywords := true
	for _, part := range tokens {
		if part.Kind != tokenizer.KindKeyword {
			allKeywords = false
			break
		}
	}
	if allKeywords {
		kind = model.ValueKindKeyword
	}

	return &model.Value{
		Kind: kind,
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

// isClauseBoundaryKeyword checks if a keyword marks a clause boundary.
func isClauseBoundaryKeyword(text string) bool {
	_, ok := clauseBoundaryKeywords[text]
	return ok
}

// clauseBoundaryKeywords defines keywords that end a clause.
var clauseBoundaryKeywords = map[string]struct{}{
	"CONSTRAINT": {},
	"PRIMARY":    {},
	"UNIQUE":     {},
	"FOREIGN":    {},
	"CHECK":      {},
	"EXCLUDE":    {},
	"REFERENCES": {},
	"DEFAULT":    {},
	"NOT":        {},
	"GENERATED":  {},
	"ON":         {},
	"DEFERRABLE": {},
	"INITIALLY":  {},
	"MATCH":      {},
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
