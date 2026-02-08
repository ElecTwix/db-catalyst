package mysql

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

	// Skip IF EXISTS
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

	// Check for COLUMN keyword (optional in MySQL)
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

// parseCreateTable handles CREATE TABLE statements with MySQL-specific syntax.
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
			if tok.Text == "CONSTRAINT" || tok.Text == "PRIMARY" || tok.Text == "UNIQUE" ||
				tok.Text == "FOREIGN" || tok.Text == "CHECK" || tok.Text == "INDEX" ||
				tok.Text == "KEY" || tok.Text == "FULLTEXT" || tok.Text == "SPATIAL" {
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

	// Parse table options (MySQL specific)
	ps.parseTableOptions(table, &span)

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

// parseColumnDefinition parses a column definition with MySQL-specific types.
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

	// Parse column type (MySQL types can be complex with attributes)
	typeStr, lastTypeTok, ok := ps.parseColumnType()
	if ok {
		res.column.Type = typeStr
		res.lastTok = lastTypeTok
	}

	// Parse column constraints and attributes
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
		case "PRIMARY":
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

		case "NULL":
			// Explicit NULL (default in MySQL)
			res.lastTok = ps.advance()

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

		case "UNIQUE":
			uniqTok := ps.advance()
			res.unique = &model.UniqueKey{Columns: []string{res.column.Name}, Span: tokenizer.NewSpan(uniqTok)}
			res.lastTok = uniqTok

		case "CHECK":
			checkTok := ps.advance()
			if last := ps.skipCheckConstraint(); last.Line != 0 {
				res.lastTok = last
			} else {
				res.lastTok = checkTok
			}
			return res, true

		case "AUTO_INCREMENT":
			aiTok := ps.advance()
			res.lastTok = aiTok
			// Mark as auto-increment via primary key if not already set
			if res.pk == nil {
				res.column.NotNull = true
				pk := &model.PrimaryKey{
					Columns: []string{res.column.Name},
					Span:    tokenizer.NewSpan(aiTok),
				}
				res.pk = pk
			}

		case "COMMENT":
			// Skip COMMENT 'string'
			ps.advance()
			if ps.current().Kind == tokenizer.KindString {
				res.lastTok = ps.advance()
			}

		case "CHARACTER", "CHARSET", "COLLATE":
			// Skip character set specifications
			ps.advance()
			if ps.current().Kind == tokenizer.KindIdentifier || ps.current().Kind == tokenizer.KindKeyword {
				res.lastTok = ps.advance()
			}

		case "UNSIGNED", "SIGNED", "ZEROFILL":
			// Skip numeric attributes
			res.lastTok = ps.advance()

		default:
			ps.addDiagToken(tok, diagnostic.SeverityWarning, "unsupported column attribute %s", tok.Text)
			res.lastTok = tok
			ps.advance()
		}
	}

	res.column.Span = tokenizer.SpanBetween(nameTok, res.lastTok)
	return res, true
}

// parseColumnType parses a MySQL column type including ENUM/SET.
func (ps *parserState) parseColumnType() (string, tokenizer.Token, bool) {
	tok := ps.current()
	if tok.Kind != tokenizer.KindIdentifier && tok.Kind != tokenizer.KindKeyword {
		return "", tok, false
	}

	typeParts := []string{tok.Text}
	lastTok := tok
	ps.advance()

	// Handle type modifiers like VARCHAR(255), DECIMAL(10,2), ENUM('a','b')
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
					if depth < 0 {
						// End of type modifier
						lastTok = t
						ps.advance()
						goto checkAttributes
					}
				}
			}
			typeParts = append(typeParts, t.Text)
			lastTok = t
			ps.advance()
		}
	}

checkAttributes:
	// Check for UNSIGNED, SIGNED, ZEROFILL
	for ps.matchKeyword("UNSIGNED") || ps.matchKeyword("SIGNED") || ps.matchKeyword("ZEROFILL") {
		attrTok := ps.advance()
		typeParts = append(typeParts, attrTok.Text)
		lastTok = attrTok
	}

	return strings.Join(typeParts, ""), lastTok, true
}

// parseTableOptions parses MySQL table options like ENGINE, CHARSET, etc.
func (ps *parserState) parseTableOptions(table *model.Table, span *tokenizer.Span) {
	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ";" {
			return
		}
		if tok.Kind != tokenizer.KindKeyword {
			if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
				// End of CREATE TABLE
				return
			}
			ps.advance()
			continue
		}

		switch tok.Text {
		case "ENGINE":
			ps.advance()
			if ps.matchSymbol("=") {
				ps.advance()
			}
			if ps.current().Kind == tokenizer.KindIdentifier || ps.current().Kind == tokenizer.KindKeyword {
				*span = span.Extend(ps.advance())
			}

		case "DEFAULT", "CHARSET", "CHARACTER", "COLLATE", "AUTO_INCREMENT":
			ps.advance()
			if ps.matchSymbol("=") {
				ps.advance()
			}
			if ps.current().Kind == tokenizer.KindIdentifier || ps.current().Kind == tokenizer.KindKeyword ||
				ps.current().Kind == tokenizer.KindNumber {
				*span = span.Extend(ps.advance())
			}

		case "COMMENT":
			ps.advance()
			if ps.matchSymbol("=") {
				ps.advance()
			}
			if ps.current().Kind == tokenizer.KindString {
				*span = span.Extend(ps.advance())
			}

		default:
			return
		}
	}
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
