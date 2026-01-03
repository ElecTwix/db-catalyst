// Package parser implements a DDL parser for SQLite schemas.
package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Severity indicates diagnostic severity.
type Severity int

const (
	// SeverityError indicates a parsing or validation error.
	SeverityError Severity = iota
	// SeverityWarning indicates a non-fatal warning.
	SeverityWarning
)

// Diagnostic captures parser feedback for callers to display.
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// Parser consumes tokenizer output and produces a normalized catalog.
type Parser struct {
	tokens []tokenizer.Token
	pos    int

	catalog     *model.Catalog
	diagnostics []Diagnostic
	pendingDoc  string
	path        string
}

// Parse constructs a catalog from the provided tokens, collecting diagnostics.
func Parse(path string, tokens []tokenizer.Token) (*model.Catalog, []Diagnostic, error) {
	p := &Parser{
		tokens:  tokens,
		catalog: model.NewCatalog(),
		path:    path,
	}
	if len(tokens) == 0 || tokens[len(tokens)-1].Kind != tokenizer.KindEOF {
		// Guarantee an EOF token to simplify parsing loops.
		p.tokens = append(p.tokens, tokenizer.Token{Kind: tokenizer.KindEOF, File: path})
	}
	if err := p.parse(); err != nil {
		return nil, p.diagnostics, err
	}
	p.validate()
	return p.catalog, p.diagnostics, nil
}

func (p *Parser) parse() error {
	for !p.isEOF() {
		tok := p.current()
		switch tok.Kind {
		case tokenizer.KindDocComment:
			p.pendingDoc = tok.Text
			p.advance()
		case tokenizer.KindKeyword:
			switch tok.Text {
			case "CREATE":
				p.parseCreate()
			case "ALTER":
				p.parseAlter()
			default:
				p.addDiagToken(tok, SeverityError, "unsupported statement starting with %s", tok.Text)
				p.sync()
			}
		case tokenizer.KindSymbol:
			if tok.Text == ";" {
				p.advance()
			} else {
				p.addDiagToken(tok, SeverityError, "unexpected symbol %q", tok.Text)
				p.advance()
			}
		case tokenizer.KindEOF:
			return nil
		default:
			p.addDiagToken(tok, SeverityError, "unexpected token %q", tok.Text)
			p.sync()
		}
	}
	return nil
}

func (p *Parser) parseCreate() {
	createTok := p.advance() // consume CREATE
	for p.matchKeyword("TEMP") || p.matchKeyword("TEMPORARY") {
		p.addDiagToken(p.current(), SeverityWarning, "TEMP/TEMPORARY modifiers are ignored")
		p.advance()
	}
	if p.matchKeyword("UNIQUE") {
		p.advance()
		if p.matchKeyword("INDEX") {
			p.advance()
			p.parseCreateIndex(createTok, true)
			return
		}
		p.addDiagToken(p.current(), SeverityError, "UNIQUE modifier only supported for indexes")
		p.sync()
		return
	}
	tok := p.current()
	if tok.Kind != tokenizer.KindKeyword {
		p.addDiagToken(tok, SeverityError, "expected TABLE, INDEX, or VIEW after CREATE")
		p.sync()
		return
	}
	switch tok.Text {
	case "TABLE":
		p.advance()
		p.parseCreateTable(createTok)
	case "INDEX":
		p.advance()
		p.parseCreateIndex(createTok, false)
	case "VIEW":
		p.advance()
		p.parseCreateView(createTok)
	default:
		p.addDiagToken(tok, SeverityError, "unsupported CREATE target %s", tok.Text)
		p.sync()
	}
}

func (p *Parser) parseCreateTable(createTok tokenizer.Token) {
	p.skipIfNotExists()
	name, nameTok, ok := p.parseObjectName()
	if !ok {
		p.sync()
		return
	}
	table := &model.Table{
		Name: name,
		Doc:  p.takeDoc(),
	}
	span := tokenizer.NewSpan(createTok)
	span = span.Extend(nameTok)
	if !p.expectSymbol("(") {
		p.sync()
		return
	}
	opener := p.previous()
	span = span.Extend(opener)
	colSeen := make(map[string]tokenizer.Span)
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
			closer := p.advance()
			span = span.Extend(closer)
			break
		}
		if tok.Kind == tokenizer.KindSymbol && tok.Text == "," {
			span = span.Extend(tok)
			p.advance()
			continue
		}
		if tok.Kind == tokenizer.KindKeyword {
			if tok.Text == "CONSTRAINT" || tok.Text == "PRIMARY" || tok.Text == "UNIQUE" || tok.Text == "FOREIGN" || tok.Text == "CHECK" {
				p.parseTableConstraint(table)
				continue
			}
		}
		res, ok := p.parseColumnDefinition()
		if !ok {
			p.skipUntilClauseEnd()
			continue
		}
		canon := canonicalName(res.column.Name)
		if prev, exists := colSeen[canon]; exists {
			p.addDiagSpan(res.column.Span, SeverityError, "duplicate column %q (previous definition at %d:%d)", res.column.Name, prev.StartLine, prev.StartColumn)
		} else {
			table.Columns = append(table.Columns, res.column)
			colSeen[canon] = res.column.Span
		}
		if res.pk != nil {
			if table.PrimaryKey != nil {
				p.addDiagSpan(res.pk.Span, SeverityError, "table %s already has a primary key", table.Name)
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
		if p.matchSymbol(",") {
			comma := p.advance()
			span = span.Extend(comma)
		}
	}
	for !p.isEOF() {
		if p.matchKeyword("WITHOUT") {
			withoutTok := p.advance()
			span = span.Extend(withoutTok)
			if p.matchKeyword("ROWID") {
				rowIDTok := p.advance()
				table.WithoutRowID = true
				span = span.Extend(rowIDTok)
			} else {
				p.addDiagToken(p.current(), SeverityError, "expected ROWID after WITHOUT")
			}
			continue
		}
		if p.matchKeyword("STRICT") {
			strictTok := p.advance()
			table.Strict = true
			span = span.Extend(strictTok)
			continue
		}
		if p.matchSymbol(",") {
			comma := p.advance()
			span = span.Extend(comma)
			continue
		}
		break
	}
	if p.matchSymbol(";") {
		semi := p.advance()
		span = span.Extend(semi)
	}
	table.Span = span
	key := canonicalName(name)
	if existing, ok := p.catalog.Tables[key]; ok {
		p.addDiagSpan(table.Span, SeverityError, "duplicate table %q (previous definition at %d:%d)", name, existing.Span.StartLine, existing.Span.StartColumn)
		return
	}
	p.catalog.Tables[key] = table
}

func (p *Parser) parseTableConstraint(table *model.Table) {
	var constraintName string
	if p.matchKeyword("CONSTRAINT") {
		p.advance()
		name, _, ok := p.parseIdentifierToken(true)
		if ok {
			constraintName = name
		}
	}
	tok := p.current()
	if tok.Kind != tokenizer.KindKeyword {
		p.addDiagToken(tok, SeverityError, "expected PRIMARY, UNIQUE, or FOREIGN after CONSTRAINT")
		p.skipUntilClauseEnd()
		return
	}
	switch tok.Text {
	case "PRIMARY":
		start := p.advance()
		if !p.matchKeyword("KEY") {
			p.addDiagToken(p.current(), SeverityError, "expected KEY after PRIMARY")
		} else {
			p.advance()
		}
		cols, last, ok := p.parseColumnNameList()
		if !ok {
			return
		}
		pk := &model.PrimaryKey{Name: constraintName, Columns: cols, Span: tokenizer.SpanBetween(start, last)}
		if table.PrimaryKey != nil {
			p.addDiagSpan(pk.Span, SeverityError, "table %s already has a primary key", table.Name)
			return
		}
		table.PrimaryKey = pk
	case "UNIQUE":
		start := p.advance()
		cols, last, ok := p.parseColumnNameList()
		if !ok {
			return
		}
		table.UniqueKeys = append(table.UniqueKeys, &model.UniqueKey{Name: constraintName, Columns: cols, Span: tokenizer.SpanBetween(start, last)})
	case "FOREIGN":
		start := p.advance()
		if !p.matchKeyword("KEY") {
			p.addDiagToken(p.current(), SeverityError, "expected KEY after FOREIGN")
			return
		}
		p.advance()
		cols, _, ok := p.parseColumnNameList()
		if !ok {
			return
		}
		if !p.matchKeyword("REFERENCES") {
			p.addDiagToken(p.current(), SeverityError, "expected REFERENCES in foreign key constraint")
			p.skipUntilClauseEnd()
			return
		}
		p.advance()
		ref, refLast, ok := p.parseForeignKeyRef()
		if !ok {
			return
		}
		fkEnd := refLast
		if extra := p.skipForeignKeyActions(); extra.Line != 0 {
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
		p.advance()
		p.skipCheckConstraint()
	default:
		p.addDiagToken(tok, SeverityError, "unsupported table constraint %s", tok.Text)
		p.skipUntilClauseEnd()
	}
}

func (p *Parser) parseCreateIndex(createTok tokenizer.Token, unique bool) {
	p.skipIfNotExists()
	name, nameTok, ok := p.parseIdentifierToken(true)
	if !ok {
		p.sync()
		return
	}
	if !p.matchKeyword("ON") {
		p.addDiagToken(p.current(), SeverityError, "expected ON in CREATE INDEX")
		p.sync()
		return
	}
	p.advance()
	tableName, _, ok := p.parseObjectName()
	if !ok {
		p.sync()
		return
	}
	cols, last, ok := p.parseColumnNameList()
	if !ok {
		return
	}
	tail := last
	if tok := p.current(); (tok.Kind == tokenizer.KindKeyword || tok.Kind == tokenizer.KindIdentifier) && strings.EqualFold(tok.Text, "WHERE") {
		whereTok := p.advance()
		tail = whereTok
		if exprLast := p.skipStatementTail(); exprLast.Line != 0 {
			tail = exprLast
		}
	}
	idx := &model.Index{
		Name:    name,
		Unique:  unique,
		Columns: cols,
		Span:    tokenizer.SpanBetween(createTok, tail),
	}
	table := p.lookupTable(tableName)
	if table == nil {
		p.addDiagSpan(idx.Span, SeverityError, "index %q references unknown table %q", name, tableName)
	} else {
		table.Indexes = append(table.Indexes, idx)
		idxSpan := tokenizer.SpanBetween(nameTok, tail)
		table.Span = table.Span.Extend(tokenizer.Token{File: idxSpan.File, Line: idxSpan.EndLine, Column: idxSpan.EndColumn})
	}
	if p.matchSymbol(";") {
		p.advance()
	}
}

func (p *Parser) parseCreateView(createTok tokenizer.Token) {
	p.skipIfNotExists()
	name, _, ok := p.parseObjectName()
	if !ok {
		p.sync()
		return
	}
	view := &model.View{
		Name: name,
		Doc:  p.takeDoc(),
	}
	if !p.matchKeyword("AS") {
		p.addDiagToken(p.current(), SeverityError, "expected AS in CREATE VIEW")
		p.sync()
		return
	}
	p.advance()
	var sqlTokens []tokenizer.Token
	last := createTok
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ";" {
			last = tok
			p.advance()
			break
		}
		sqlTokens = append(sqlTokens, tok)
		last = tok
		p.advance()
	}
	view.SQL = rebuildSQL(sqlTokens)
	view.Span = tokenizer.SpanBetween(createTok, last)
	key := canonicalName(name)
	if _, exists := p.catalog.Views[key]; exists {
		p.addDiagSpan(view.Span, SeverityError, "duplicate view %q", name)
		return
	}
	p.catalog.Views[key] = view
}

func (p *Parser) parseAlter() {
	alterTok := p.advance()
	if !p.matchKeyword("TABLE") {
		p.addDiagToken(p.current(), SeverityError, "expected TABLE after ALTER")
		p.sync()
		return
	}
	p.advance()
	tableName, nameTok, ok := p.parseObjectName()
	if !ok {
		p.sync()
		return
	}
	table := p.lookupTable(tableName)
	if table == nil {
		p.addDiagToken(nameTok, SeverityError, "ALTER TABLE references unknown table %q", tableName)
	}
	if !p.matchKeyword("ADD") {
		p.addDiagToken(p.current(), SeverityError, "expected ADD in ALTER TABLE")
		p.sync()
		return
	}
	p.advance()
	if p.matchKeyword("COLUMN") {
		p.advance()
	}
	res, ok := p.parseColumnDefinition()
	if !ok {
		p.sync()
		return
	}
	if table == nil {
		return
	}
	canon := canonicalName(res.column.Name)
	if p.tableHasColumn(table, canon) {
		p.addDiagSpan(res.column.Span, SeverityError, "table %s already has column %q", table.Name, res.column.Name)
		return
	}
	table.Columns = append(table.Columns, res.column)
	if res.pk != nil {
		if table.PrimaryKey != nil {
			p.addDiagSpan(res.pk.Span, SeverityError, "table %s already has a primary key", table.Name)
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
	table.Span = table.Span.Extend(alterTok)
	if p.matchSymbol(";") {
		p.advance()
	}
}

type columnResult struct {
	column  *model.Column
	pk      *model.PrimaryKey
	unique  *model.UniqueKey
	foreign *model.ForeignKey
	lastTok tokenizer.Token
}

func (p *Parser) parseColumnDefinition() (*columnResult, bool) {
	name, nameTok, ok := p.parseIdentifierToken(true)
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
	typeParts := make([]string, 0, 2)
	for {
		tok := p.current()
		if tok.Kind == tokenizer.KindIdentifier || tok.Kind == tokenizer.KindKeyword {
			if _, isConstraint := columnConstraintStarters[tok.Text]; isConstraint {
				break
			}
			typeParts = append(typeParts, tok.Text)
			res.lastTok = tok
			p.advance()
			continue
		}
		break
	}
	if len(typeParts) > 0 {
		res.column.Type = strings.Join(typeParts, " ")
	}
	for {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")" || tok.Text == ";") {
			break
		}
		if tok.Kind == tokenizer.KindEOF {
			p.addDiagToken(tok, SeverityError, "unexpected EOF in column definition for %s", res.column.Name)
			return res, false
		}
		if tok.Kind != tokenizer.KindKeyword {
			res.lastTok = tok
			p.advance()
			continue
		}
		switch tok.Text {
		case "PRIMARY":
			start := p.advance()
			if p.matchKeyword("KEY") {
				keyTok := p.advance()
				res.lastTok = keyTok
			} else {
				p.addDiagToken(p.current(), SeverityError, "expected KEY after PRIMARY")
			}
			res.column.NotNull = true
			pk := &model.PrimaryKey{Columns: []string{res.column.Name}, Span: tokenizer.NewSpan(start)}
			if res.lastTok.Line != 0 {
				pk.Span = tokenizer.SpanBetween(start, res.lastTok)
			}
			res.pk = pk
			if p.matchKeyword("AUTOINCREMENT") {
				autoTok := p.advance()
				res.lastTok = autoTok
			}
		case "NOT":
			p.advance()
			if p.matchKeyword("NULL") {
				nullTok := p.advance()
				res.column.NotNull = true
				res.lastTok = nullTok
			} else {
				p.addDiagToken(p.current(), SeverityError, "expected NULL after NOT")
			}

		case "DEFAULT":
			p.advance()
			val, last := p.parseDefaultValue()
			if val != nil {
				res.column.Default = val
				res.lastTok = last
			}
		case "REFERENCES":
			referTok := p.advance()
			ref, last, ok := p.parseForeignKeyRef()
			if ok {
				fkEnd := last
				if extra := p.skipForeignKeyActions(); extra.Line != 0 {
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
			uniqTok := p.advance()
			res.unique = &model.UniqueKey{Columns: []string{res.column.Name}, Span: tokenizer.NewSpan(uniqTok)}
			res.lastTok = uniqTok
		case "CHECK":
			checkTok := p.advance()
			if last := p.skipCheckConstraint(); last.Line != 0 {
				res.lastTok = last
			} else {
				res.lastTok = checkTok
			}
			return res, true
		default:
			p.addDiagToken(tok, SeverityWarning, "unsupported column constraint %s", tok.Text)
			res.lastTok = tok
			p.advance()
		}
	}
	res.column.Span = tokenizer.SpanBetween(nameTok, res.lastTok)
	return res, true
}

func (p *Parser) parseForeignKeyRef() (*model.ForeignKeyRef, tokenizer.Token, bool) {
	name, nameTok, ok := p.parseObjectName()
	if !ok {
		return nil, nameTok, false
	}
	ref := &model.ForeignKeyRef{
		Table: name,
		Span:  tokenizer.NewSpan(nameTok),
	}
	last := nameTok
	if p.matchSymbol("(") {
		cols, closeTok, ok := p.parseColumnNameList()
		if !ok {
			return nil, closeTok, false
		}
		ref.Columns = cols
		last = closeTok
	}
	ref.Span = tokenizer.SpanBetween(nameTok, last)
	return ref, last, true
}

func (p *Parser) parseColumnNameList() ([]string, tokenizer.Token, bool) {
	if !p.matchSymbol("(") {
		p.addDiagToken(p.current(), SeverityError, "expected ( to start column list")
		return nil, p.current(), false
	}
	open := p.advance()
	var names []string
	last := open
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ")" {
			last = p.advance()
			return names, last, true
		}
		name, nameTok, ok := p.parseIdentifierToken(true)
		if !ok {
			return names, nameTok, false
		}
		names = append(names, name)
		last = nameTok
		last = p.skipColumnOrderingTokens(last)
		if p.matchSymbol(",") {
			last = p.advance()
			continue
		}
	}
	p.addDiagToken(p.current(), SeverityError, "unexpected EOF in column list")
	return names, last, false
}

func (p *Parser) parseDefaultValue() (*model.Value, tokenizer.Token) {
	tok := p.current()
	switch tok.Kind {
	case tokenizer.KindNumber:
		p.advance()
		return &model.Value{Kind: model.ValueKindNumber, Text: tok.Text, Span: tokenizer.NewSpan(tok)}, tok
	case tokenizer.KindString:
		p.advance()
		return &model.Value{Kind: model.ValueKindString, Text: tok.Text, Span: tokenizer.NewSpan(tok)}, tok
	case tokenizer.KindBlob:
		p.advance()
		return &model.Value{Kind: model.ValueKindBlob, Text: tok.Text, Span: tokenizer.NewSpan(tok)}, tok
	}
	tokens, last := p.collectDefaultExpressionTokens()
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
	} else if len(tokens) == 1 && (tokens[0].Kind == tokenizer.KindKeyword || tokens[0].Kind == tokenizer.KindIdentifier) {
		kind = model.ValueKindKeyword
	}
	text := rebuildSQL(tokens)
	span := tokenizer.SpanBetween(tokens[0], last)
	return &model.Value{Kind: kind, Text: text, Span: span}, last
}
func (p *Parser) collectDefaultExpressionTokens() ([]tokenizer.Token, tokenizer.Token) {
	var tokens []tokenizer.Token
	var last tokenizer.Token
	depth := 0
	for !p.isEOF() {
		tok := p.current()
		if len(tokens) == 0 {
			if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")" || tok.Text == ";") {
				break
			}
			if tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
				break
			}
		} else if depth == 0 {
			if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")" || tok.Text == ";") {
				break
			}
			if tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
				break
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
			}
		}
		tokens = append(tokens, tok)
		last = tok
		p.advance()
	}
	return tokens, last
}

func (p *Parser) skipColumnOrderingTokens(last tokenizer.Token) tokenizer.Token {
	depth := 0
	for !p.isEOF() {
		tok := p.current()
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
		last = tok
		p.advance()
	}
	return last
}

func (p *Parser) skipBalancedParentheses() tokenizer.Token {
	if !p.matchSymbol("(") {
		return tokenizer.Token{}
	}
	open := p.advance()
	depth := 1
	last := open
	for !p.isEOF() && depth > 0 {
		tok := p.current()
		last = tok
		if tok.Kind == tokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				depth--
			}
		}
		p.advance()
	}
	return last
}

func (p *Parser) skipCheckConstraint() tokenizer.Token {
	var last tokenizer.Token
	if p.matchSymbol("(") {
		last = p.skipBalancedParentheses()
	}
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")") {
			break
		}
		if tok.Kind == tokenizer.KindKeyword && isClauseBoundaryKeyword(tok.Text) {
			break
		}
		last = tok
		p.advance()
	}
	return last
}

func (p *Parser) skipForeignKeyActions() tokenizer.Token {
	var last tokenizer.Token
	depth := 0
	for !p.isEOF() {
		tok := p.current()
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
		p.advance()
	}
	return last
}

func (p *Parser) skipStatementTail() tokenizer.Token {
	var last tokenizer.Token
	depth := 0
	for !p.isEOF() {
		tok := p.current()
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
		p.advance()
	}
	return last
}

func isClauseBoundaryKeyword(text string) bool {
	_, ok := clauseBoundaryKeywords[text]
	return ok
}

var clauseBoundaryKeywords = map[string]struct{}{
	"CONSTRAINT": {},
	"PRIMARY":    {},
	"UNIQUE":     {},
	"FOREIGN":    {},
	"CHECK":      {},
	"REFERENCES": {},
	"DEFAULT":    {},
	"NOT":        {},
	"GENERATED":  {},
}

func (p *Parser) validate() {
	tableKeys := make([]string, 0, len(p.catalog.Tables))
	for key := range p.catalog.Tables {
		tableKeys = append(tableKeys, key)
	}
	slices.Sort(tableKeys)
	for _, key := range tableKeys {
		table := p.catalog.Tables[key]
		p.validateTable(table)
	}
	viewKeys := make([]string, 0, len(p.catalog.Views))
	for key := range p.catalog.Views {
		viewKeys = append(viewKeys, key)
	}
	slices.Sort(viewKeys)
}

func (p *Parser) validateTable(table *model.Table) {
	colSet := make(map[string]struct{}, len(table.Columns))
	for _, col := range table.Columns {
		colSet[canonicalName(col.Name)] = struct{}{}
	}
	if table.PrimaryKey != nil {
		for _, name := range table.PrimaryKey.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				p.addDiagSpan(table.PrimaryKey.Span, SeverityError, "primary key references unknown column %q on table %s", name, table.Name)
			}
		}
	}
	for _, uk := range table.UniqueKeys {
		for _, name := range uk.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				p.addDiagSpan(uk.Span, SeverityError, "unique constraint references unknown column %q on table %s", name, table.Name)
			}
		}
	}
	for _, fk := range table.ForeignKeys {
		for _, name := range fk.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				p.addDiagSpan(fk.Span, SeverityError, "foreign key references unknown column %q on table %s", name, table.Name)
			}
		}
		if fk.Ref.Table == "" {
			continue
		}
		refTable := p.lookupTable(fk.Ref.Table)
		if refTable == nil {
			p.addDiagSpan(fk.Span, SeverityError, "foreign key references unknown table %q", fk.Ref.Table)
			continue
		}
		if len(fk.Ref.Columns) == 0 {
			continue
		}
		refCols := make(map[string]struct{})
		for _, col := range refTable.Columns {
			refCols[canonicalName(col.Name)] = struct{}{}
		}
		for _, col := range fk.Ref.Columns {
			if _, ok := refCols[canonicalName(col)]; !ok {
				p.addDiagSpan(fk.Span, SeverityError, "foreign key references unknown column %q on table %s", col, refTable.Name)
			}
		}
	}
	for _, idx := range table.Indexes {
		for _, col := range idx.Columns {
			if _, ok := colSet[canonicalName(col)]; !ok {
				p.addDiagSpan(idx.Span, SeverityError, "index %q references unknown column %q on table %s", idx.Name, col, table.Name)
			}
		}
	}
}

func (p *Parser) addDiagToken(tok tokenizer.Token, severity Severity, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	diag := Diagnostic{
		Path:     tok.File,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
		Severity: severity,
	}
	if diag.Path == "" {
		diag.Path = p.path
	}
	if diag.Line == 0 {
		diag.Line = 1
	}
	if diag.Column == 0 {
		diag.Column = 1
	}
	p.diagnostics = append(p.diagnostics, diag)
}

func (p *Parser) addDiagSpan(span tokenizer.Span, severity Severity, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	diag := Diagnostic{
		Path:     span.File,
		Line:     span.StartLine,
		Column:   span.StartColumn,
		Message:  msg,
		Severity: severity,
	}
	if diag.Path == "" {
		diag.Path = p.path
	}
	if diag.Line == 0 {
		diag.Line = 1
	}
	if diag.Column == 0 {
		diag.Column = 1
	}
	p.diagnostics = append(p.diagnostics, diag)
}

func (p *Parser) skipIfNotExists() {
	if p.matchKeyword("IF") {
		p.advance()
		if p.matchKeyword("NOT") {
			p.advance()
		}
		if p.matchKeyword("EXISTS") {
			p.advance()
		}
	}
}

func (p *Parser) parseObjectName() (string, tokenizer.Token, bool) {
	name, tok, ok := p.parseIdentifierToken(true)
	if !ok {
		return "", tok, false
	}
	if p.matchSymbol(".") {
		p.advance()
		alias, aliasTok, ok := p.parseIdentifierToken(true)
		if ok {
			return alias, aliasTok, true
		}
	}
	return name, tok, true
}

func (p *Parser) parseIdentifierToken(allowKeyword bool) (string, tokenizer.Token, bool) {
	tok := p.current()
	if tok.Kind == tokenizer.KindIdentifier || (allowKeyword && tok.Kind == tokenizer.KindKeyword) {
		p.advance()
		name := tokenizer.NormalizeIdentifier(tok.Text)
		return name, tok, true
	}
	p.addDiagToken(tok, SeverityError, "expected identifier, got %s", tok.Kind.String())
	return "", tok, false
}

func (p *Parser) matchKeyword(text string) bool {
	tok := p.current()
	return tok.Kind == tokenizer.KindKeyword && tok.Text == text
}

func (p *Parser) matchSymbol(text string) bool {
	tok := p.current()
	return tok.Kind == tokenizer.KindSymbol && tok.Text == text
}

func (p *Parser) expectSymbol(text string) bool {
	if !p.matchSymbol(text) {
		p.addDiagToken(p.current(), SeverityError, "expected symbol %q", text)
		return false
	}
	p.advance()
	return true
}

func (p *Parser) previous() tokenizer.Token {
	if p.pos == 0 {
		return tokenizer.Token{}
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) current() tokenizer.Token {
	if p.pos >= len(p.tokens) {
		return tokenizer.Token{Kind: tokenizer.KindEOF, File: p.path}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() tokenizer.Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) isEOF() bool {
	return p.current().Kind == tokenizer.KindEOF
}

func (p *Parser) sync() {
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ";" {
			p.advance()
			return
		}
		if tok.Kind == tokenizer.KindKeyword && (tok.Text == "CREATE" || tok.Text == "ALTER") {
			return
		}
		p.advance()
	}
}

func (p *Parser) skipUntilClauseEnd() {
	for !p.isEOF() {
		tok := p.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")") {
			return
		}
		p.advance()
	}
}

func (p *Parser) tableHasColumn(table *model.Table, canon string) bool {
	for _, col := range table.Columns {
		if canonicalName(col.Name) == canon {
			return true
		}
	}
	return false
}

func (p *Parser) lookupTable(name string) *model.Table {
	return p.catalog.Tables[canonicalName(name)]
}

func (p *Parser) takeDoc() string {
	doc := p.pendingDoc
	p.pendingDoc = ""
	return strings.TrimSpace(doc)
}

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

func canonicalName(name string) string {
	return strings.ToLower(name)
}

var columnConstraintStarters = map[string]struct{}{
	"PRIMARY":    {},
	"NOT":        {},
	"DEFAULT":    {},
	"REFERENCES": {},
	"CHECK":      {},
	"UNIQUE":     {},
	"CONSTRAINT": {},
}
