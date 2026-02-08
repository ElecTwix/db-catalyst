package mysql

import (
	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// parseTableConstraint parses table-level constraints.
func (ps *parserState) parseTableConstraint(table *model.Table) {
	var constraintName string
	if ps.matchKeyword(KeywordConstraint) {
		ps.advance()
		name, _, ok := ps.parseIdentifier(true)
		if ok {
			constraintName = name
		}
	}

	tok := ps.current()
	if tok.Kind != tokenizer.KindKeyword {
		ps.addDiagToken(tok, diagnostic.SeverityError, "expected PRIMARY, UNIQUE, FOREIGN, INDEX, or KEY")
		ps.skipUntilClauseEnd()
		return
	}

	switch tok.Text {
	case KeywordPrimary:
		start := ps.advance()
		if !ps.matchKeyword(KeywordKey) {
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

	case KeywordUnique:
		start := ps.advance()
		// UNIQUE can be followed by INDEX/KEY
		if ps.matchKeyword(KeywordIndex) || ps.matchKeyword(KeywordKey) {
			ps.advance()
		}
		// Optional index name
		if ps.current().Kind == tokenizer.KindIdentifier {
			ps.advance()
		}
		cols, last, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		table.UniqueKeys = append(table.UniqueKeys, &model.UniqueKey{
			Name:    constraintName,
			Columns: cols,
			Span:    tokenizer.SpanBetween(start, last),
		})

	case KeywordIndex, KeywordKey:
		start := ps.advance()
		// Optional index name
		indexName := ""
		if ps.current().Kind == tokenizer.KindIdentifier {
			name, nameTok, _ := ps.parseIdentifier(true)
			indexName = name
			_ = nameTok
		}
		cols, last, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		table.Indexes = append(table.Indexes, &model.Index{
			Name:    indexName,
			Columns: cols,
			Span:    tokenizer.SpanBetween(start, last),
		})
		_ = last

	case KeywordFullText:
		start := ps.advance()
		// Optional INDEX/KEY
		if ps.matchKeyword(KeywordIndex) || ps.matchKeyword(KeywordKey) {
			ps.advance()
		}
		// Optional index name
		if ps.current().Kind == tokenizer.KindIdentifier {
			ps.advance()
		}
		cols, last, ok := ps.parseColumnNameList()
		if !ok {
			return
		}
		table.Indexes = append(table.Indexes, &model.Index{
			Name:    constraintName,
			Columns: cols,
			Span:    tokenizer.SpanBetween(start, last),
		})
		_ = last

	case KeywordForeign:
		start := ps.advance()
		if !ps.matchKeyword(KeywordKey) {
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

	case KeywordCheck:
		ps.advance()
		ps.skipCheckConstraint()

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

		// Skip length specification like VARCHAR(255) or column attributes
		last = ps.skipColumnAttributes(last)

		if ps.matchSymbol(",") {
			last = ps.advance()
			continue
		}
	}

	ps.addDiagToken(ps.current(), diagnostic.SeverityError, "unexpected EOF in column list")
	return names, last, false
}

// skipColumnAttributes skips column attributes like length, ASC, DESC.
func (ps *parserState) skipColumnAttributes(last tokenizer.Token) tokenizer.Token {
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

		// Skip ASC, DESC keywords
		if tok.Kind == tokenizer.KindKeyword {
			switch tok.Text {
			case "ASC", "DESC":
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

// skipBalancedParentheses skips balanced parentheses.
func (ps *parserState) skipBalancedParentheses() tokenizer.Token {
	if !ps.matchSymbol("(") {
		return tokenizer.Token{}
	}
	ps.advance()
	depth := 1
	var last tokenizer.Token

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

// isClauseBoundaryKeyword checks if a keyword marks a clause boundary.
func isClauseBoundaryKeyword(text string) bool {
	_, ok := clauseBoundaryKeywords[text]
	return ok
}

// clauseBoundaryKeywords defines keywords that end a clause.
var clauseBoundaryKeywords = map[string]struct{}{
	KeywordConstraint:     {},
	KeywordPrimary:        {},
	KeywordUnique:         {},
	KeywordForeign:        {},
	KeywordIndex:          {},
	KeywordKey:            {},
	KeywordFullText:       {},
	KeywordCheck:          {},
	"REFERENCES":     {},
	"DEFAULT":        {},
	"NOT":            {},
	"NULL":           {},
	"COMMENT":        {},
	"AUTO_INCREMENT": {},
	"ON":             {},
	"ENGINE":         {},
	"CHARSET":        {},
	"CHARACTER":      {},
	"COLLATE":        {},
}
