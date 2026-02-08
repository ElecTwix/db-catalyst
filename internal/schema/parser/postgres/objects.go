package postgres

import (
	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// parseCreateIndex handles CREATE INDEX statements.
func (ps *parserState) parseCreateIndex(unique bool) {
	createTok := ps.previous()

	// Skip IF NOT EXISTS
	ps.skipIfNotExists()

	// Parse index name
	name, nameTok, ok := ps.parseIdentifier(true)
	if !ok {
		ps.sync()
		return
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

	// Parse USING clause (optional, e.g., USING GIN, USING GIST)
	if ps.matchKeyword("USING") {
		ps.advance()
		// Skip the index method name
		ps.parseIdentifier(true)
	}

	// Parse column list
	cols, last, ok := ps.parseColumnNameList()
	if !ok {
		return
	}

	tail := last

	// Parse WHERE clause (optional for partial indexes)
	if ps.matchKeyword("WHERE") {
		whereTok := ps.advance()
		tail = whereTok
		if exprLast := ps.skipStatementTail(); exprLast.Line != 0 {
			tail = exprLast
		}
	}

	idx := &model.Index{
		Name:    name,
		Unique:  unique,
		Columns: cols,
		Span:    tokenizer.SpanBetween(createTok, tail),
	}

	table := ps.lookupTable(tableName)
	if table == nil {
		ps.addDiagSpan(idx.Span, diagnostic.SeverityError,
			"index %q references unknown table %q", name, tableName)
	} else {
		table.Indexes = append(table.Indexes, idx)
		idxSpan := tokenizer.SpanBetween(nameTok, tail)
		table.Span = table.Span.Extend(tokenizer.Token{
			File:   idxSpan.File,
			Line:   idxSpan.EndLine,
			Column: idxSpan.EndColumn,
		})
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

	// Skip TEMP/TEMPORARY
	if ps.matchKeyword("TEMP") || ps.matchKeyword("TEMPORARY") {
		ps.advance()
	}

	// Skip IF NOT EXISTS
	ps.skipIfNotExists()

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

// parseCreateType handles CREATE TYPE statements for enums and composite types.
func (ps *parserState) parseCreateType() {
	// Parse type name
	name, _, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	// Check for AS
	if !ps.matchKeyword("AS") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected AS in CREATE TYPE")
		ps.sync()
		return
	}
	ps.advance()

	// Check for ENUM
	if ps.matchKeyword("ENUM") {
		ps.advance()
		ps.parseEnumType(name)
		return
	}

	// Check for RANGE
	if ps.matchKeyword("RANGE") {
		ps.advance()
		// Skip range definition
		ps.skipBalancedParentheses()
		return
	}

	// Composite type - parse column definitions
	if ps.matchSymbol("(") {
		ps.skipBalancedParentheses()
	}
}

// parseEnumType parses an enum type definition.
func (ps *parserState) parseEnumType(name string) {
	if !ps.expectSymbol("(") {
		return
	}

	var values []string
	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
			ps.advance()
			break
		}

		if tok.Kind == tokenizer.KindString {
			values = append(values, tok.Text)
			ps.advance()
		} else {
			ps.addDiagToken(tok, diagnostic.SeverityError, "expected string literal for enum value")
			ps.advance()
		}

		if ps.matchSymbol(",") {
			ps.advance()
		}
	}

	// Store enum in catalog
	// TODO: Add Enums field to model.Catalog for v0.5.0
	_ = name
	_ = values

	if ps.matchSymbol(";") {
		ps.advance()
	}
}

// parseCreateDomain handles CREATE DOMAIN statements.
func (ps *parserState) parseCreateDomain() {
	// Parse domain name
	name, _, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	// Expect AS
	if !ps.matchKeyword("AS") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected AS in CREATE DOMAIN")
		ps.sync()
		return
	}
	ps.advance()

	// Parse base type
	baseType, _, _ := ps.parseColumnType()

	// Parse constraints
	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == ";" || tok.Text == ")") {
			break
		}

		if tok.Kind == tokenizer.KindKeyword {
			switch tok.Text {
			case "DEFAULT":
				ps.advance()
				ps.parseDefaultValue()
			case "NOT":
				ps.advance()
				if ps.matchKeyword("NULL") {
					ps.advance()
				}
			case "CHECK":
				ps.advance()
				ps.skipCheckConstraint()
			case "CONSTRAINT":
				ps.advance()
				// Skip constraint name
				ps.parseIdentifier(true)
			default:
				ps.advance()
			}
		} else {
			ps.advance()
		}
	}

	// Store domain in catalog
	// TODO: Add Domains field to model.Catalog for v0.5.0
	_ = name
	_ = baseType

	if ps.matchSymbol(";") {
		ps.advance()
	}
}

// skipStatementTail skips the rest of a statement.
func (ps *parserState) skipStatementTail() tokenizer.Token {
	var last tokenizer.Token
	depth := 0

	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			case ";":
				if depth == 0 {
					return last
				}
			}
		}
		last = tok
		ps.advance()
	}

	return last
}
