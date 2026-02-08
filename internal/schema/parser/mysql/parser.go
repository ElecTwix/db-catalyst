// Package mysql implements a DDL parser for MySQL schemas.
//
// This parser supports MySQL-specific DDL syntax including:
//   - AUTO_INCREMENT
//   - TIMESTAMP with ON UPDATE
//   - ENUM and SET types
//   - Full-text indexes
//   - MySQL-specific column attributes
package mysql

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Parser implements diagnostic.SchemaParser for MySQL dialect.
type Parser struct {
	keywords map[string]struct{}
}

// New creates a new MySQL schema parser.
func New() *Parser {
	return &Parser{
		keywords: mysqlKeywords(),
	}
}

// Parse parses MySQL DDL content and returns a catalog.
func (p *Parser) Parse(ctx context.Context, path string, content []byte) (*model.Catalog, []diagnostic.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("parse cancelled: %w", err)
	}

	tokens, err := tokenizer.Scan(path, content, true)
	if err != nil {
		return nil, nil, fmt.Errorf("tokenization failed: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("parse cancelled: %w", err)
	}

	return p.parse(path, tokens)
}

// parserState holds the current parsing state.
type parserState struct {
	tokens []tokenizer.Token
	pos    int

	catalog     *model.Catalog
	diagnostics []diagnostic.Diagnostic
	pendingDoc  string
	path        string
}

// parse constructs a catalog from the provided tokens.
func (p *Parser) parse(path string, tokens []tokenizer.Token) (*model.Catalog, []diagnostic.Diagnostic, error) {
	ps := &parserState{
		tokens:  tokens,
		catalog: model.NewCatalog(),
		path:    path,
	}

	if len(tokens) == 0 || tokens[len(tokens)-1].Kind != tokenizer.KindEOF {
		ps.tokens = append(ps.tokens, tokenizer.Token{Kind: tokenizer.KindEOF, File: path})
	}

	if err := ps.parse(); err != nil {
		return nil, ps.diagnostics, err
	}

	ps.validate()
	return ps.catalog, ps.diagnostics, nil
}

// parse is the main parsing loop.
func (ps *parserState) parse() error {
	for !ps.isEOF() {
		tok := ps.current()
		switch tok.Kind {
		case tokenizer.KindDocComment:
			ps.pendingDoc = tok.Text
			ps.advance()
		case tokenizer.KindKeyword:
			switch tok.Text {
			case "CREATE":
				ps.advance()
				ps.parseCreate()
			case "ALTER":
				ps.advance()
				ps.parseAlter()
			default:
				ps.addDiagToken(tok, diagnostic.SeverityError, "unsupported statement starting with %s", tok.Text)
				ps.sync()
			}
		case tokenizer.KindSymbol:
			if tok.Text == ";" {
				ps.advance()
			} else {
				ps.addDiagToken(tok, diagnostic.SeverityError, "unexpected symbol %q", tok.Text)
				ps.advance()
			}
		case tokenizer.KindEOF:
			return nil
		default:
			ps.addDiagToken(tok, diagnostic.SeverityError, "unexpected token %q", tok.Text)
			ps.sync()
		}
	}
	return nil
}

// parseCreate handles CREATE statements.
func (ps *parserState) parseCreate() {
	// Check for TEMPORARY
	if ps.matchKeyword("TEMPORARY") {
		ps.advance()
	}

	isUnique := false
	if ps.matchKeyword("UNIQUE") {
		isUnique = true
		ps.advance()
	}

	tok := ps.current()
	if tok.Kind != tokenizer.KindKeyword {
		ps.addDiagToken(tok, diagnostic.SeverityError, "expected TABLE, INDEX, or VIEW after CREATE")
		ps.sync()
		return
	}

	switch tok.Text {
	case "TABLE":
		ps.advance()
		ps.parseCreateTable()
	case "INDEX":
		ps.advance()
		ps.parseCreateIndex(isUnique)
	case "VIEW":
		ps.advance()
		ps.parseCreateView()
	default:
		ps.addDiagToken(tok, diagnostic.SeverityError, "unsupported CREATE target %s", tok.Text)
		ps.sync()
	}
}

// mysqlKeywords returns MySQL-specific keywords.
func mysqlKeywords() map[string]struct{} {
	kw := map[string]struct{}{
		"ACTION":            {},
		"ADD":               {},
		"AFTER":             {},
		"ALGORITHM":         {},
		"ALTER":             {},
		"ANALYZE":           {},
		"ASC":               {},
		"AUTO_INCREMENT":    {},
		"BIGINT":            {},
		"BINARY":            {},
		"BLOB":              {},
		"BOOL":              {},
		"BOOLEAN":           {},
		"BY":                {},
		"CASCADE":           {},
		"CHAR":              {},
		"CHARACTER":         {},
		"CHECK":             {},
		"COLLATE":           {},
		"COLUMN":            {},
		"COMMENT":           {},
		"CONSTRAINT":        {},
		"CREATE":            {},
		"CURRENT_TIMESTAMP": {},
		"DATABASE":          {},
		"DATE":              {},
		"DATETIME":          {},
		"DEC":               {},
		"DECIMAL":           {},
		"DEFAULT":           {},
		"DELETE":            {},
		"DESC":              {},
		"DISTINCT":          {},
		"DOUBLE":            {},
		"DROP":              {},
		"ELSE":              {},
		"ENUM":              {},
		"EXISTS":            {},
		"FALSE":             {},
		"FLOAT":             {},
		"FOREIGN":           {},
		"FROM":              {},
		"FULLTEXT":          {},
		"IF":                {},
		"IGNORE":            {},
		"INDEX":             {},
		"INNER":             {},
		"INNODB":            {},
		"INSERT":            {},
		"INT":               {},
		"INTEGER":           {},
		"INTO":              {},
		"JOIN":              {},
		"JSON":              {},
		"KEY":               {},
		"KEYS":              {},
		"LEFT":              {},
		"LIKE":              {},
		"LIMIT":             {},
		"LOCK":              {},
		"LONGBLOB":          {},
		"LONGTEXT":          {},
		"MATCH":             {},
		"MEDIUMBLOB":        {},
		"MEDIUMINT":         {},
		"MEDIUMTEXT":        {},
		"MODIFY":            {},
		"NOT":               {},
		"NULL":              {},
		"NUMERIC":           {},
		"ON":                {},
		"OR":                {},
		"ORDER":             {},
		"OUTER":             {},
		"PRIMARY":           {},
		"REAL":              {},
		"REFERENCES":        {},
		"RENAME":            {},
		"REPLACE":           {},
		"RESTRICT":          {},
		"RIGHT":             {},
		"SCHEMA":            {},
		"SELECT":            {},
		"SET":               {},
		"SIGNED":            {},
		"SMALLINT":          {},
		"SPATIAL":           {},
		"SQL":               {},
		"TABLE":             {},
		"TEMPORARY":         {},
		"TEXT":              {},
		"THEN":              {},
		"TIME":              {},
		"TIMESTAMP":         {},
		"TINYBLOB":          {},
		"TINYINT":           {},
		"TINYTEXT":          {},
		"TO":                {},
		"TRUE":              {},
		"UNIQUE":            {},
		"UNSIGNED":          {},
		"UPDATE":            {},
		"USING":             {},
		"VARBINARY":         {},
		"VARCHAR":           {},
		"VIEW":              {},
		"WHEN":              {},
		"WHERE":             {},
		"WITH":              {},
		"YEAR":              {},
		"ZEROFILL":          {},
	}
	return kw
}

// canonicalName normalizes names to lowercase for consistent lookup.
func canonicalName(name string) string {
	return strings.ToLower(name)
}

// isEOF reports whether we've reached the end of input.
func (ps *parserState) isEOF() bool {
	return ps.current().Kind == tokenizer.KindEOF
}

// current returns the current token.
func (ps *parserState) current() tokenizer.Token {
	if ps.pos >= len(ps.tokens) {
		return tokenizer.Token{Kind: tokenizer.KindEOF, File: ps.path}
	}
	return ps.tokens[ps.pos]
}

// advance consumes and returns the current token.
func (ps *parserState) advance() tokenizer.Token {
	tok := ps.current()
	if ps.pos < len(ps.tokens) {
		ps.pos++
	}
	return tok
}

// previous returns the previously consumed token.
func (ps *parserState) previous() tokenizer.Token {
	if ps.pos == 0 {
		return tokenizer.Token{}
	}
	return ps.tokens[ps.pos-1]
}

// matchKeyword reports whether the current token is the specified keyword.
func (ps *parserState) matchKeyword(text string) bool {
	tok := ps.current()
	return tok.Kind == tokenizer.KindKeyword && tok.Text == text
}

// matchSymbol reports whether the current token is the specified symbol.
func (ps *parserState) matchSymbol(text string) bool {
	tok := ps.current()
	return tok.Kind == tokenizer.KindSymbol && tok.Text == text
}

// expectSymbol consumes the symbol if it matches, otherwise reports an error.
func (ps *parserState) expectSymbol(text string) bool {
	if !ps.matchSymbol(text) {
		ps.addDiagToken(ps.current(), diagnostic.SeverityError, "expected symbol %q", text)
		return false
	}
	ps.advance()
	return true
}

// addDiagToken adds a diagnostic for a token.
func (ps *parserState) addDiagToken(tok tokenizer.Token, severity diagnostic.Severity, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	diag := diagnostic.Diagnostic{
		Path:     tok.File,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
		Severity: severity,
	}
	if diag.Path == "" {
		diag.Path = ps.path
	}
	if diag.Line == 0 {
		diag.Line = 1
	}
	if diag.Column == 0 {
		diag.Column = 1
	}
	ps.diagnostics = append(ps.diagnostics, diag)
}

// addDiagSpan adds a diagnostic for a span.
func (ps *parserState) addDiagSpan(span tokenizer.Span, severity diagnostic.Severity, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	diag := diagnostic.Diagnostic{
		Path:     span.File,
		Line:     span.StartLine,
		Column:   span.StartColumn,
		Message:  msg,
		Severity: severity,
	}
	if diag.Path == "" {
		diag.Path = ps.path
	}
	if diag.Line == 0 {
		diag.Line = 1
	}
	if diag.Column == 0 {
		diag.Column = 1
	}
	ps.diagnostics = append(ps.diagnostics, diag)
}

// sync skips tokens until a safe resync point.
func (ps *parserState) sync() {
	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && tok.Text == ";" {
			ps.advance()
			return
		}
		if tok.Kind == tokenizer.KindKeyword && (tok.Text == "CREATE" || tok.Text == "ALTER") {
			return
		}
		ps.advance()
	}
}

// skipUntilClauseEnd skips tokens until clause boundary.
func (ps *parserState) skipUntilClauseEnd() {
	for !ps.isEOF() {
		tok := ps.current()
		if tok.Kind == tokenizer.KindSymbol && (tok.Text == "," || tok.Text == ")") {
			return
		}
		ps.advance()
	}
}

// takeDoc retrieves and clears the pending documentation comment.
func (ps *parserState) takeDoc() string {
	doc := ps.pendingDoc
	ps.pendingDoc = ""
	return strings.TrimSpace(doc)
}

// parseObjectName parses a potentially backtick-quoted or bare identifier.
func (ps *parserState) parseObjectName() (string, tokenizer.Token, bool) {
	return ps.parseIdentifier(true)
}

// parseIdentifier parses an identifier token.
func (ps *parserState) parseIdentifier(allowKeyword bool) (string, tokenizer.Token, bool) {
	tok := ps.current()
	if tok.Kind == tokenizer.KindIdentifier || (allowKeyword && tok.Kind == tokenizer.KindKeyword) {
		ps.advance()
		name := tokenizer.NormalizeIdentifier(tok.Text)
		return name, tok, true
	}
	ps.addDiagToken(tok, diagnostic.SeverityError, "expected identifier, got %s", tok.Kind.String())
	return "", tok, false
}

// lookupTable finds a table by name in the catalog.
func (ps *parserState) lookupTable(name string) *model.Table {
	return ps.catalog.Tables[canonicalName(name)]
}

// tableHasColumn checks if a table has a column.
func (ps *parserState) tableHasColumn(table *model.Table, canon string) bool {
	for _, col := range table.Columns {
		if canonicalName(col.Name) == canon {
			return true
		}
	}
	return false
}

// validate performs post-parse validation.
func (ps *parserState) validate() {
	tableKeys := slices.Collect(maps.Keys(ps.catalog.Tables))
	slices.Sort(tableKeys)

	for _, key := range tableKeys {
		table := ps.catalog.Tables[key]
		ps.validateTable(table)
	}
}

// validateTable validates a single table.
func (ps *parserState) validateTable(table *model.Table) {
	colSet := make(map[string]struct{}, len(table.Columns))
	for _, col := range table.Columns {
		colSet[canonicalName(col.Name)] = struct{}{}
	}

	if table.PrimaryKey != nil {
		for _, name := range table.PrimaryKey.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				ps.addDiagSpan(table.PrimaryKey.Span, diagnostic.SeverityError,
					"primary key references unknown column %q on table %s", name, table.Name)
			}
		}
	}

	for _, uk := range table.UniqueKeys {
		for _, name := range uk.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				ps.addDiagSpan(uk.Span, diagnostic.SeverityError,
					"unique constraint references unknown column %q on table %s", name, table.Name)
			}
		}
	}

	for _, fk := range table.ForeignKeys {
		for _, name := range fk.Columns {
			if _, ok := colSet[canonicalName(name)]; !ok {
				ps.addDiagSpan(fk.Span, diagnostic.SeverityError,
					"foreign key references unknown column %q on table %s", name, table.Name)
			}
		}

		if fk.Ref.Table == "" {
			continue
		}

		refTable := ps.lookupTable(fk.Ref.Table)
		if refTable == nil {
			ps.addDiagSpan(fk.Span, diagnostic.SeverityError,
				"foreign key references unknown table %q", fk.Ref.Table)
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
				ps.addDiagSpan(fk.Span, diagnostic.SeverityError,
					"foreign key references unknown column %q on table %s", col, refTable.Name)
			}
		}
	}

	for _, idx := range table.Indexes {
		for _, col := range idx.Columns {
			if _, ok := colSet[canonicalName(col)]; !ok {
				ps.addDiagSpan(idx.Span, diagnostic.SeverityError,
					"index %q references unknown column %q on table %s", idx.Name, col, table.Name)
			}
		}
	}
}
