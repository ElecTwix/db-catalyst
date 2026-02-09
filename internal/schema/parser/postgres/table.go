package postgres

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// parseAlter handles ALTER statements.
func (ps *parserState) parseAlter() {
	if !ps.matchKeyword("TABLE") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected TABLE after ALTER")
		ps.sync()
		return
	}
	ps.advance()

	// Check for IF EXISTS
	ps.skipIfExists()

	tableName, nameTok, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	table := ps.lookupTable(tableName)
	if table == nil {
		ps.addDiagToken(nameTok, diagnostic.SeverityError, "ALTER TABLE references unknown table %q", tableName)
	}

	if !ps.matchKeyword("ADD") {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected ADD in ALTER TABLE")
		ps.sync()
		return
	}
	ps.advance()

	// Check for COLUMN keyword (optional in PostgreSQL)
	if ps.matchKeyword("COLUMN") {
		ps.advance()
	}

	res, ok := ps.parseColumnDefinition()
	if !ok {
		ps.sync()
		return
	}

	if table == nil {
		return
	}

	canon := canonicalName(res.column.Name)
	if ps.tableHasColumn(table, canon) {
		ps.addDiagSpan(res.column.Span, diagnostic.SeverityError,
			"table %s already has column %q", table.Name, res.column.Name)
		return
	}

	table.Columns = append(table.Columns, res.column)

	if res.pk != nil {
		if table.PrimaryKey != nil {
			ps.addDiagSpan(res.pk.Span, diagnostic.SeverityError,
				"table %s already has a primary key", table.Name)
		} else {
			table.PrimaryKey = res.pk
		}
	}

	if res.unique != nil {
		table.UniqueKeys = append(table.UniqueKeys, res.unique)
	}

	if res.foreign != nil {
		table.ForeignKeys = append(table.ForeignKeys, res.foreign)
	}

	if ps.matchSymbol(";") {
		ps.advance()
	}
}

// parseCreateTable handles CREATE TABLE statements with PostgreSQL-specific syntax.
func (ps *parserState) parseCreateTable() {
	createTok := ps.previous()

	// Check for IF NOT EXISTS
	ps.skipIfNotExists()

	name, nameTok, ok := ps.parseObjectName()
	if !ok {
		ps.sync()
		return
	}

	table := &model.Table{
		Name: name,
		Doc:  ps.takeDoc(),
	}

	span := tokenizer.NewSpan(createTok)
	span = span.Extend(nameTok)

	if !ps.expectSymbol("(") {
		ps.sync()
		return
	}
	opener := ps.previous()
	span = span.Extend(opener)

	colSeen := make(map[string]tokenizer.Span)

	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
			closer := ps.advance()
			span = span.Extend(closer)
			break
		}

		if tok.Kind == tokenizer.KindSymbol && tok.Text == "," {
			span = span.Extend(tok)
			ps.advance()
			continue
		}

		// Check for table-level constraints
		if tok.Kind == tokenizer.KindKeyword {
			if tok.Text == KeywordConstraint || tok.Text == KeywordPrimary || tok.Text == KeywordUnique ||
				tok.Text == KeywordForeign || tok.Text == KeywordCheck || tok.Text == KeywordExclude {
				ps.parseTableConstraint(table)
				continue
			}
		}

		res, ok := ps.parseColumnDefinition()
		if !ok {
			ps.skipUntilClauseEnd()
			continue
		}

		canon := canonicalName(res.column.Name)
		if prev, exists := colSeen[canon]; exists {
			ps.addDiagSpan(res.column.Span, diagnostic.SeverityError,
				"duplicate column %q (previous definition at %d:%d)",
				res.column.Name, prev.StartLine, prev.StartColumn)
		} else {
			table.Columns = append(table.Columns, res.column)
			colSeen[canon] = res.column.Span
		}

		if res.pk != nil {
			if table.PrimaryKey != nil {
				ps.addDiagSpan(res.pk.Span, diagnostic.SeverityError,
					"table %s already has a primary key", table.Name)
			} else {
				table.PrimaryKey = res.pk
			}
		}

		if res.unique != nil {
			table.UniqueKeys = append(table.UniqueKeys, res.unique)
		}

		if res.foreign != nil {
			table.ForeignKeys = append(table.ForeignKeys, res.foreign)
		}

		span = span.Extend(res.lastTok)

		if ps.matchSymbol(",") {
			comma := ps.advance()
			span = span.Extend(comma)
		}
	}

	// Parse table-level options (PostgreSQL has fewer than SQLite)
	if ps.matchSymbol(";") {
		semi := ps.advance()
		span = span.Extend(semi)
	}

	table.Span = span

	key := canonicalName(name)
	if existing, ok := ps.catalog.Tables[key]; ok {
		ps.addDiagSpan(table.Span, diagnostic.SeverityError,
			"duplicate table %q (previous definition at %d:%d)",
			name, existing.Span.StartLine, existing.Span.StartColumn)
		return
	}

	ps.catalog.Tables[key] = table
}

// columnResult holds the result of parsing a column definition.
type columnResult struct {
	column  *model.Column
	pk      *model.PrimaryKey
	unique  *model.UniqueKey
	foreign *model.ForeignKey
	lastTok tokenizer.Token
}

// parseColumnDefinition parses a column definition with PostgreSQL-specific types.
func (ps *parserState) parseColumnDefinition() (*columnResult, bool) {
	name, nameTok, ok := ps.parseIdentifier(true)
	if !ok {
		return nil, false
	}

	res := &columnResult{
		column: &model.Column{
			Name: name,
			Span: tokenizer.NewSpan(nameTok),
		},
		lastTok: nameTok,
	}

	// Parse column type (PostgreSQL types can be complex)
	typeStr, lastTypeTok, ok := ps.parseColumnType()
	if ok {
		// Handle multi-word PostgreSQL types
		if typeStr == "DOUBLE" && ps.matchKeyword("PRECISION") {
			precisionTok := ps.advance()
			typeStr = "DOUBLE PRECISION"
			lastTypeTok = precisionTok
		}
		res.column.Type = typeStr
		res.lastTok = lastTypeTok
	}

	// Parse column constraints
	for {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")" || tok.Text == ";") {
			break
		}
		if tok.Kind == tokenizer.KindEOF {
			ps.addDiagToken(tok, diagnostic.SeverityError,
				"unexpected EOF in column definition for %s", res.column.Name)
			return res, false
		}
		if tok.Kind != tokenizer.KindKeyword {
			res.lastTok = tok
			ps.advance()
			continue
		}

		switch tok.Text {
		case KeywordPrimary:
			start := ps.advance()
			if ps.matchKeyword("KEY") {
				keyTok := ps.advance()
				res.lastTok = keyTok
			} else {
				ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected KEY after PRIMARY")
			}
			res.column.NotNull = true
			pk := &model.PrimaryKey{Columns: []string{res.column.Name}, Span: tokenizer.NewSpan(start)}
			if res.lastTok.Line != 0 {
				pk.Span = tokenizer.SpanBetween(start, res.lastTok)
			}
			res.pk = pk

		case "NOT":
			ps.advance()
			if ps.matchKeyword("NULL") {
				nullTok := ps.advance()
				res.column.NotNull = true
				res.lastTok = nullTok
			} else {
				ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected NULL after NOT")
			}

		case "DEFAULT":
			ps.advance()
			val, last := ps.parseDefaultValue()
			if val != nil {
				res.column.Default = val
				res.lastTok = last
			}

		case "REFERENCES":
			referTok := ps.advance()
			ref, last, ok := ps.parseForeignKeyRef()
			if ok {
				fkEnd := last
				if extra := ps.skipForeignKeyActions(); extra.Line != 0 {
					fkEnd = extra
				}
				res.column.References = ref
				res.foreign = &model.ForeignKey{
					Columns: []string{res.column.Name},
					Ref:     *ref,
					Span:    tokenizer.SpanBetween(referTok, fkEnd),
				}
				res.lastTok = fkEnd
			}

		case KeywordUnique:
			uniqTok := ps.advance()
			res.unique = &model.UniqueKey{Columns: []string{res.column.Name}, Span: tokenizer.NewSpan(uniqTok)}
			res.lastTok = uniqTok

		case KeywordCheck:
			checkTok := ps.advance()
			if last := ps.skipCheckConstraint(); last.Line != 0 {
				res.lastTok = last
			} else {
				res.lastTok = checkTok
			}
			return res, true

		case "GENERATED":
			// GENERATED ALWAYS AS IDENTITY or GENERATED BY DEFAULT AS IDENTITY
			genTok := ps.advance()
			res.lastTok = genTok
			if ps.matchKeyword("ALWAYS") || ps.matchKeyword("BY") {
				ps.advance()
				if ps.matchKeyword("DEFAULT") {
					ps.advance()
				}
				if ps.matchKeyword("AS") {
					ps.advance()
					if ps.matchKeyword("IDENTITY") {
						idTok := ps.advance()
						res.lastTok = idTok
						// Mark as auto-increment via primary key
						res.column.NotNull = true
						pk := &model.PrimaryKey{
							Columns: []string{res.column.Name},
							Span:    tokenizer.SpanBetween(genTok, idTok),
						}
						res.pk = pk
					}
				}
			}

		default:
			ps.addDiagToken(tok, diagnostic.SeverityWarning, "unsupported column constraint %s", tok.Text)
			res.lastTok = tok
			ps.advance()
		}
	}

	res.column.Span = tokenizer.SpanBetween(nameTok, res.lastTok)
	return res, true
}

// parseColumnType parses a PostgreSQL column type including arrays and modifiers.
func (ps *parserState) parseColumnType() (string, tokenizer.Token, bool) {
	tok := ps.current()
	if tok.Kind != tokenizer.KindIdentifier && tok.Kind != tokenizer.KindKeyword {
		return "", tok, false
	}

	typeParts := []string{tok.Text}
	lastTok := tok
	ps.advance()

	// Handle type modifiers like VARCHAR(255), NUMERIC(10,2)
	if ps.matchSymbol("(") {
		depth := 0
		for !ps.isEOF() {
			t := ps.current()
			if t.Kind == tokenizer.KindSymbol {
				switch t.Text {
				case "(":
					depth++
				case ")":
					depth--
					if depth <= 0 {
						// End of type modifier - include the closing paren
						typeParts = append(typeParts, t.Text)
						lastTok = t
						ps.advance()
						goto checkArray
					}
				}
			}
			typeParts = append(typeParts, t.Text)
			lastTok = t
			ps.advance()
		}
	}

checkArray:
	// Check for array type modifier []
	// Handle both separate [ ] tokens and combined [] identifier
	if ps.matchSymbol("[") {
		bracket := ps.advance()
		typeParts = append(typeParts, bracket.Text)
		if ps.matchSymbol("]") {
			closeBracket := ps.advance()
			typeParts = append(typeParts, closeBracket.Text)
			lastTok = closeBracket
		}
	} else if ps.current().Kind == tokenizer.KindIdentifier && ps.current().Text == "[]" {
		// Tokenizer combined [] into a single identifier
		bracket := ps.advance()
		typeParts = append(typeParts, bracket.Text)
		lastTok = bracket
	}

	return strings.Join(typeParts, ""), lastTok, true
}

// skipIfNotExists skips IF NOT EXISTS clause.
func (ps *parserState) skipIfNotExists() {
	if ps.matchKeyword("IF") {
		ps.advance()
		if ps.matchKeyword("NOT") {
			ps.advance()
		}
		if ps.matchKeyword("EXISTS") {
			ps.advance()
		}
	}
}

// skipIfExists skips IF EXISTS clause.
func (ps *parserState) skipIfExists() {
	if ps.matchKeyword("IF") {
		ps.advance()
		if ps.matchKeyword("EXISTS") {
			ps.advance()
		}
	}
}
