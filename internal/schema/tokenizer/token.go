package tokenizer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind represents the classification of a scanned token.
type Kind int

const (
	// KindInvalid represents an unrecognized or placeholder token.
	KindInvalid Kind = iota
	// KindIdentifier represents bare or quoted identifiers.
	KindIdentifier
	// KindKeyword represents SQL keywords normalized to uppercase.
	KindKeyword
	// KindNumber represents numeric literals.
	KindNumber
	// KindString represents string literals using single quotes.
	KindString
	// KindBlob represents blob literals of the form X'...'.
	KindBlob
	// KindSymbol represents punctuation or operator symbols.
	KindSymbol
	// KindParam represents PostgreSQL-style positional parameters ($1, $2, etc.)
	KindParam
	// KindDocComment represents a documentation comment captured for a following statement.
	KindDocComment
	// KindEOF marks the logical end of the input.
	KindEOF
)

// Token is a unit emitted by the scanner with positional metadata.
type Token struct {
	Kind   Kind
	Text   string
	File   string
	Line   int
	Column int
}

// Span represents a best-effort start and end position within a source file.
type Span struct {
	File        string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

// NewSpan returns a span covering a single token.
func NewSpan(tok Token) Span {
	return Span{
		File:        tok.File,
		StartLine:   tok.Line,
		StartColumn: tok.Column,
		EndLine:     tok.Line,
		EndColumn:   spanEndColumn(tok),
	}
}

// SpanBetween returns a span that covers both the start and end tokens, inclusive.
func SpanBetween(start, end Token) Span {
	span := NewSpan(start)
	if end.File != "" {
		span.File = end.File
	}
	span.EndLine = end.Line
	span.EndColumn = spanEndColumn(end)
	if span.EndLine < span.StartLine || (span.EndLine == span.StartLine && span.EndColumn < span.StartColumn) {
		span.EndLine = span.StartLine
		span.EndColumn = span.StartColumn
	}
	return span
}

// Extend expands the span to include the provided token.
func (s Span) Extend(tok Token) Span {
	if s.StartLine == 0 && s.StartColumn == 0 {
		return NewSpan(tok)
	}
	if tok.File != "" {
		s.File = tok.File
	}
	if tok.Line < s.StartLine || (tok.Line == s.StartLine && tok.Column < s.StartColumn) {
		s.StartLine = tok.Line
		s.StartColumn = tok.Column
	}
	endLine := tok.Line
	endColumn := spanEndColumn(tok)
	if endLine > s.EndLine || (endLine == s.EndLine && endColumn > s.EndColumn) {
		s.EndLine = endLine
		s.EndColumn = endColumn
	}
	return s
}

func spanEndColumn(tok Token) int {
	width := utf8.RuneCountInString(tok.Text)
	if width <= 0 {
		return tok.Column
	}
	return tok.Column + width
}

// Error describes a positional scanning error suitable for diagnostics.
type Error struct {
	Path    string
	Line    int
	Column  int
	Message string
}

// Error returns the printable representation of the tokenizer error.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Path != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
}

// IsKeyword reports whether the provided string matches a known keyword.
func IsKeyword(s string) bool {
	if s == "" {
		return false
	}
	_, ok := keywords[strings.ToUpper(s)]
	return ok
}

// NormalizeIdentifier removes optional quoting from identifiers while unescaping content.
func NormalizeIdentifier(text string) string {
	if len(text) < 2 {
		return text
	}
	switch text[0] {
	case '"':
		if text[len(text)-1] != '"' {
			return text
		}
		inner := strings.ReplaceAll(text[1:len(text)-1], "\"\"", "\"")
		return inner
	case '[':
		if text[len(text)-1] != ']' {
			return text
		}
		return text[1 : len(text)-1]
	case '`':
		if text[len(text)-1] != '`' {
			return text
		}
		return text[1 : len(text)-1]
	default:
		return text
	}
}

var keywords = map[string]struct{}{
	"ABORT":         {},
	"ACTION":        {},
	"ADD":           {},
	"AFTER":         {},
	"ALL":           {},
	"ALTER":         {},
	"AND":           {},
	"AS":            {},
	"ASC":           {},
	"AUTOINCREMENT": {},
	"BEFORE":        {},
	"BLOB":          {},
	"BOOLEAN":       {},
	"CASCADE":       {},
	"CHAR":          {},
	"CHECK":         {},
	"COLUMN":        {},
	"CONFLICT":      {},
	"CONSTRAINT":    {},
	"CREATE":        {},
	"CROSS":         {},
	"CURRENT":       {},
	"DEFAULT":       {},
	"DEFERRABLE":    {},
	"DEFERRED":      {},
	"DELETE":        {},
	"DESC":          {},
	"DISTINCT":      {},
	"DOUBLE":        {},
	"DROP":          {},
	"EACH":          {},
	"ELSE":          {},
	"EXISTS":        {},
	"FOREIGN":       {},
	"FROM":          {},
	"IF":            {},
	"IMMEDIATE":     {},
	"INDEX":         {},
	"INITIALLY":     {},
	"INNER":         {},
	"INSERT":        {},
	"INTEGER":       {},
	"INSTEAD":       {},
	"INTO":          {},
	"JOIN":          {},
	"KEY":           {},
	"LEFT":          {},
	"MATCH":         {},
	"NO":            {},
	"NOT":           {},
	"NULL":          {},
	"NUMERIC":       {},
	"OF":            {},
	"ON":            {},
	"OR":            {},
	"OUTER":         {},
	"PRECISION":     {},
	"PRIMARY":       {},
	"REAL":          {},
	"REFERENCES":    {},
	"RENAME":        {},
	"REPLACE":       {},
	"RESTRICT":      {},
	"RETURNING":     {},
	"ROLLBACK":      {},
	"ROW":           {},
	"ROWID":         {},
	"SET":           {},
	"STRICT":        {},
	"TABLE":         {},
	"TEMP":          {},
	"TEMPORARY":     {},
	"TEXT":          {},
	"TRIGGER":       {},
	"UNIQUE":        {},
	"UPDATE":        {},
	"USING":         {},
	"VALUES":        {},
	"VARCHAR":       {},
	"VIEW":          {},
	"VIRTUAL":       {},
	"WITHOUT":       {},
}

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "Invalid"
	case KindIdentifier:
		return "Identifier"
	case KindKeyword:
		return "Keyword"
	case KindNumber:
		return "Number"
	case KindString:
		return "String"
	case KindBlob:
		return "Blob"
	case KindSymbol:
		return "Symbol"
	case KindParam:
		return "Param"
	case KindDocComment:
		return "DocComment"
	case KindEOF:
		return "EOF"
	default:
		return "Kind(" + strconv.Itoa(int(k)) + ")"
	}
}
