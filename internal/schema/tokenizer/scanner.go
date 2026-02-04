// Package tokenizer scans SQL source code into tokens.
package tokenizer

import (
	"fmt"
	"iter"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eofRune = -1

// Scan tokenizes the provided schema source and returns the token stream.
func Scan(path string, src []byte, captureDocs bool) ([]Token, error) {
	if !utf8.Valid(src) {
		return nil, &Error{Path: path, Line: 1, Column: 1, Message: "input is not valid UTF-8"}
	}
	scanner := &Scanner{
		path:        path,
		src:         string(src),
		captureDocs: captureDocs,
		tokens:      make([]Token, 0, len(src)/4+1),
		line:        1,
		column:      1,
	}
	if err := scanner.scan(); err != nil {
		return nil, err
	}
	return scanner.tokens, nil
}

// ScanSeq returns an iterator over tokens in the source.
// This is memory-efficient for large files and enables early termination.
// Use this when you only need to process tokens sequentially.
// For random access, use Scan() instead.
//
// Example:
//
//	for tok := range tokenizer.ScanSeq(path, src, false) {
//	    if tok.Kind == tokenizer.KindEOF {
//	        break
//	    }
//	    process(tok)
//	}
func ScanSeq(path string, src []byte, captureDocs bool) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		if !utf8.Valid(src) {
			return
		}

		scanner := &scannerIter{
			path:        path,
			src:         string(src),
			captureDocs: captureDocs,
			line:        1,
			column:      1,
		}

		for scanner.index < len(scanner.src) {
			tok := scanner.nextToken()
			if !yield(tok) {
				return
			}
			if tok.Kind == KindEOF {
				return
			}
		}

		// Yield EOF
		yield(Token{Kind: KindEOF, Line: scanner.line, Column: scanner.column})
	}
}

// Scanner maintains scanning state over a schema source.
type Scanner struct {
	path        string
	src         string
	captureDocs bool
	tokens      []Token
	index       int
	line        int
	column      int
	pendingDoc  *docBuffer
}

type docBuffer struct {
	lines []string
	line  int
	col   int
}

// scannerIter is a lightweight scanner for iterator-based tokenization.
type scannerIter struct {
	path        string
	src         string
	captureDocs bool
	index       int
	line        int
	column      int
}

// nextToken returns the next token from the source.
func (s *scannerIter) nextToken() Token {
	for s.index < len(s.src) {
		r := s.peek()
		switch {
		case r == eofRune:
			s.index = len(s.src)
			return s.emitEOF()
		case unicode.IsSpace(r):
			s.consumeWhitespace()
		case r == '-' && s.peekNext() == '-':
			s.consumeLineComment()
		case r == '/' && s.peekNext() == '*':
			s.consumeBlockComment()
		case r == '\'':
			return s.consumeStringLiteral()
		case (r == 'x' || r == 'X') && s.peekNext() == '\'':
			return s.consumeBlobLiteral()
		case r == '"' || r == '[' || r == '`':
			return s.consumeQuotedIdentifier()
		case r == '$' && isDigit(s.peekNext()):
			return s.consumePostgresParam()
		case isIdentifierStart(r):
			return s.consumeIdentifier()
		case isDigit(r):
			return s.consumeNumber()
		case isSymbolRune(r):
			return s.consumeSymbol()
		default:
			startLine, startCol := s.line, s.column
			s.advance()
			return s.newToken(KindSymbol, string(r), startLine, startCol)
		}
	}
	return s.emitEOF()
}

func (s *scannerIter) emitEOF() Token {
	return Token{Kind: KindEOF, Line: s.line, Column: s.column}
}

func (s *scannerIter) newToken(kind Kind, text string, line, col int) Token {
	return Token{
		Kind:   kind,
		Text:   text,
		File:   s.path,
		Line:   line,
		Column: col,
	}
}

func (s *scannerIter) peek() rune {
	if s.index >= len(s.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(s.src[s.index:])
	return r
}

func (s *scannerIter) peekNext() rune {
	if s.index >= len(s.src) {
		return eofRune
	}
	_, size := utf8.DecodeRuneInString(s.src[s.index:])
	nextPos := s.index + size
	if nextPos >= len(s.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(s.src[nextPos:])
	return r
}

func (s *scannerIter) advance() {
	if s.index >= len(s.src) {
		return
	}
	_, size := utf8.DecodeRuneInString(s.src[s.index:])
	s.index += size
	s.column++
}

func (s *scannerIter) consumeWhitespace() {
	for {
		r := s.peek()
		if r == eofRune || !unicode.IsSpace(r) {
			return
		}
		if r == '\n' {
			s.line++
			s.column = 0
		}
		s.advance()
	}
}

func (s *scannerIter) consumeLineComment() {
	for {
		r := s.peek()
		if r == eofRune || r == '\n' {
			return
		}
		s.advance()
	}
}

func (s *scannerIter) consumeBlockComment() {
	s.advance() // '/'
	s.advance() // '*'
	for {
		r := s.peek()
		if r == eofRune {
			return
		}
		if r == '*' && s.peekNext() == '/' {
			s.advance() // '*'
			s.advance() // '/'
			return
		}
		if r == '\n' {
			s.line++
			s.column = 0
		}
		s.advance()
	}
}

func (s *scannerIter) consumeStringLiteral() Token {
	startLine, startCol := s.line, s.column
	s.advance() // opening quote
	var content strings.Builder
	for {
		r := s.peek()
		if r == eofRune {
			return s.newToken(KindString, content.String(), startLine, startCol)
		}
		if r == '\'' {
			s.advance()
			if s.peek() == '\'' {
				s.advance()
				content.WriteRune('\'')
			} else {
				return s.newToken(KindString, content.String(), startLine, startCol)
			}
		} else {
			content.WriteRune(r)
			s.advance()
		}
	}
}

func (s *scannerIter) consumeBlobLiteral() Token {
	startLine, startCol := s.line, s.column
	s.advance() // 'x' or 'X'
	s.advance() // opening quote
	var content strings.Builder
	for {
		r := s.peek()
		if r == eofRune || r == '\'' {
			if r == '\'' {
				s.advance()
			}
			return s.newToken(KindBlob, content.String(), startLine, startCol)
		}
		content.WriteRune(r)
		s.advance()
	}
}

func (s *scannerIter) consumeQuotedIdentifier() Token {
	startLine, startCol := s.line, s.column
	quote := s.peek()
	s.advance() // opening quote
	var content strings.Builder
	for {
		r := s.peek()
		if r == eofRune {
			return s.newToken(KindIdentifier, content.String(), startLine, startCol)
		}
		if r == quote {
			s.advance()
			return s.newToken(KindIdentifier, content.String(), startLine, startCol)
		}
		content.WriteRune(r)
		s.advance()
	}
}

func (s *scannerIter) consumePostgresParam() Token {
	startLine, startCol := s.line, s.column
	s.advance() // '$'
	var num strings.Builder
	for isDigit(s.peek()) {
		num.WriteRune(s.peek())
		s.advance()
	}
	return s.newToken(KindParam, "$"+num.String(), startLine, startCol)
}

func (s *scannerIter) consumeIdentifier() Token {
	startLine, startCol := s.line, s.column
	var content strings.Builder
	for isIdentifierPart(s.peek()) {
		content.WriteRune(s.peek())
		s.advance()
	}
	text := content.String()
	kind := KindIdentifier
	if IsKeyword(strings.ToUpper(text)) {
		kind = KindKeyword
	}
	return s.newToken(kind, text, startLine, startCol)
}

func (s *scannerIter) consumeNumber() Token {
	startLine, startCol := s.line, s.column
	var content strings.Builder
	for isDigit(s.peek()) || s.peek() == '.' {
		content.WriteRune(s.peek())
		s.advance()
	}
	return s.newToken(KindNumber, content.String(), startLine, startCol)
}

func (s *scannerIter) consumeSymbol() Token {
	startLine, startCol := s.line, s.column
	r := s.peek()
	s.advance()
	return s.newToken(KindSymbol, string(r), startLine, startCol)
}

func (s *Scanner) scan() error {
	for s.index < len(s.src) {
		r := s.peek()
		switch {
		case r == eofRune:
			s.index = len(s.src)
		case unicode.IsSpace(r):
			s.consumeWhitespace()
		case r == '-' && s.peekNext() == '-':
			if err := s.consumeLineComment(); err != nil {
				return err
			}
		case r == '/' && s.peekNext() == '*':
			if err := s.consumeBlockComment(); err != nil {
				return err
			}
		case r == '\'':
			if err := s.consumeStringLiteral(); err != nil {
				return err
			}
		case (r == 'x' || r == 'X') && s.peekNext() == '\'':
			if err := s.consumeBlobLiteral(); err != nil {
				return err
			}
		case r == '"' || r == '[' || r == '`':
			if err := s.consumeQuotedIdentifier(); err != nil {
				return err
			}
		case r == '$' && isDigit(s.peekNext()):
			s.consumePostgresParam()
		case isIdentifierStart(r):
			s.consumeIdentifier()
		case isDigit(r):
			s.consumeNumber()
		case isSymbolRune(r):
			s.consumeSymbol()
		default:
			startLine, startCol := s.line, s.column
			s.advance()
			s.emitToken(KindSymbol, string(r), startLine, startCol)
		}
	}
	s.emitToken(KindEOF, "", s.line, s.column)
	return nil
}

func (s *Scanner) consumeWhitespace() {
	for {
		r := s.peek()
		if r == eofRune || !unicode.IsSpace(r) {
			return
		}
		s.advance()
	}
}

func (s *Scanner) consumeLineComment() error {
	startLine, startCol := s.line, s.column
	s.advance() // first '-'
	s.advance() // second '-'
	commentStart := s.index
	for {
		r := s.peek()
		if r == eofRune || r == '\n' || r == '\r' {
			break
		}
		s.advance()
	}
	if s.captureDocs {
		raw := s.src[commentStart:s.index]
		s.recordDocLine(raw, startLine, startCol)
	}
	return nil
}

func (s *Scanner) consumeBlockComment() error {
	startLine, startCol := s.line, s.column
	s.advance() // '/'
	s.advance() // '*'
	contentStart := s.index
	for {
		if s.index >= len(s.src) {
			return s.errorf(startLine, startCol, "unterminated block comment")
		}
		r := s.peek()
		if r == '*' && s.peekNext() == '/' {
			contentEnd := s.index
			s.advance() // '*'
			s.advance() // '/'
			if s.captureDocs {
				raw := s.src[contentStart:contentEnd]
				s.recordDocBlock(raw, startLine, startCol)
			}
			return nil
		}
		s.advance()
	}
}

func (s *Scanner) consumeStringLiteral() error {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	s.advance() // opening quote
	for {
		if s.index >= len(s.src) {
			return s.errorf(startLine, startCol, "unterminated string literal")
		}
		r := s.peek()
		s.advance()
		if r == '\'' {
			if s.peek() == '\'' {
				s.advance()
				continue
			}
			break
		}
	}
	text := s.src[startIdx:s.index]
	s.emitToken(KindString, text, startLine, startCol)
	return nil
}

func (s *Scanner) consumeBlobLiteral() error {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	s.advance() // X or x
	s.advance() // opening quote
	for {
		if s.index >= len(s.src) {
			return s.errorf(startLine, startCol, "unterminated blob literal")
		}
		r := s.peek()
		s.advance()
		if r == '\'' {
			if s.peek() == '\'' {
				s.advance()
				continue
			}
			break
		}
	}
	text := s.src[startIdx:s.index]
	if len(text) < 3 {
		return s.errorf(startLine, startCol, "unterminated blob literal")
	}
	payload := text[2 : len(text)-1]
	if len(payload)%2 != 0 {
		return s.errorf(startLine, startCol, "blob literal must contain even number of hex digits")
	}
	for i := 0; i < len(payload); i++ {
		if !isHexDigit(rune(payload[i])) {
			return s.errorf(startLine, startCol, "blob literal contains non-hex digit")
		}
	}
	if text[0] == 'x' {
		text = "X" + text[1:]
	}
	s.emitToken(KindBlob, text, startLine, startCol)
	return nil
}

func (s *Scanner) consumeNumber() {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	s.advanceDigits()
	if s.peek() == '.' {
		s.advance()
		s.advanceDigits()
	}
	next := s.peek()
	if next == 'e' || next == 'E' {
		s.advance()
		signe := s.peek()
		if signe == '+' || signe == '-' {
			s.advance()
		}
		s.advanceDigits()
	}
	text := s.src[startIdx:s.index]
	s.emitToken(KindNumber, text, startLine, startCol)
}

func (s *Scanner) consumePostgresParam() {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	s.advance() // consume '$'
	s.advanceDigits()
	text := s.src[startIdx:s.index]
	s.emitToken(KindParam, text, startLine, startCol)
}

func (s *Scanner) consumeIdentifier() {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	s.advance()
	for {
		r := s.peek()
		if !isIdentifierPart(r) {
			break
		}
		s.advance()
	}
	text := s.src[startIdx:s.index]
	upper := strings.ToUpper(text)
	if IsKeyword(upper) {
		s.emitToken(KindKeyword, upper, startLine, startCol)
		return
	}
	s.emitToken(KindIdentifier, text, startLine, startCol)
}

func (s *Scanner) consumeQuotedIdentifier() error {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	quote := s.peek()
	var closing rune
	switch quote {
	case '[':
		closing = ']'
	default:
		closing = quote
	}
	s.advance() // opening quote
	for {
		if s.index >= len(s.src) {
			return s.errorf(startLine, startCol, "unterminated quoted identifier")
		}
		r := s.peek()
		s.advance()
		if r == closing {
			next := s.peek()
			if (quote == '"' && next == '"') || (quote == '[' && next == ']') || (quote == '`' && next == '`') {
				s.advance()
				continue
			}
			break
		}
	}
	text := s.src[startIdx:s.index]
	s.emitToken(KindIdentifier, text, startLine, startCol)
	return nil
}

func (s *Scanner) consumeSymbol() {
	startIdx := s.index
	startLine, startCol := s.line, s.column
	first := s.advance()
	switch first {
	case '<', '>', '!', '=':
		next := s.peek()
		if next == '=' || (first == '<' && next == '>') {
			s.advance()
		}
	case ':':
		if s.peek() == ':' {
			s.advance()
		}
	}
	text := s.src[startIdx:s.index]
	s.emitToken(KindSymbol, text, startLine, startCol)
}

func (s *Scanner) advanceDigits() {
	for isDigit(s.peek()) {
		s.advance()
	}
}

func (s *Scanner) emitToken(kind Kind, text string, line, column int) {
	if kind == KindKeyword && text == "CREATE" {
		s.emitPendingDoc()
	} else if kind != KindDocComment && kind != KindEOF {
		s.pendingDoc = nil
	}
	tok := Token{
		Kind:   kind,
		Text:   text,
		File:   s.path,
		Line:   line,
		Column: column,
	}
	s.tokens = append(s.tokens, tok)
}

func (s *Scanner) emitPendingDoc() {
	if s.pendingDoc == nil {
		return
	}
	text := strings.TrimSpace(strings.Join(s.pendingDoc.lines, "\n"))
	if text == "" {
		s.pendingDoc = nil
		return
	}
	tok := Token{
		Kind:   KindDocComment,
		Text:   text,
		File:   s.path,
		Line:   s.pendingDoc.line,
		Column: s.pendingDoc.col,
	}
	s.tokens = append(s.tokens, tok)
	s.pendingDoc = nil
}

func (s *Scanner) recordDocLine(raw string, line, column int) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
	if trimmed == "" && s.pendingDoc == nil {
		return
	}
	if s.pendingDoc == nil {
		s.pendingDoc = &docBuffer{line: line, col: column}
	}
	s.pendingDoc.lines = append(s.pendingDoc.lines, trimmed)
}

func (s *Scanner) recordDocBlock(raw string, line, column int) {
	clean := strings.ReplaceAll(raw, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return
	}
	if s.pendingDoc == nil {
		s.pendingDoc = &docBuffer{line: line, col: column}
	}
	for part := range strings.SplitSeq(clean, "\n") {
		s.pendingDoc.lines = append(s.pendingDoc.lines, strings.TrimSpace(part))
	}
}

func (s *Scanner) peek() rune {
	if s.index >= len(s.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(s.src[s.index:])
	return r
}

func (s *Scanner) peekNext() rune {
	idx := s.index
	if idx >= len(s.src) {
		return eofRune
	}
	_, size := utf8.DecodeRuneInString(s.src[idx:])
	idx += size
	if idx >= len(s.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(s.src[idx:])
	return r
}

func (s *Scanner) advance() rune {
	if s.index >= len(s.src) {
		return eofRune
	}
	r, size := utf8.DecodeRuneInString(s.src[s.index:])
	s.index += size
	switch r {
	case '\r':
		if s.index < len(s.src) && s.src[s.index] == '\n' {
			s.index++
		}
		s.line++
		s.column = 1
		return '\n'
	case '\n':
		s.line++
		s.column = 1
	default:
		s.column++
	}
	return r
}

func (s *Scanner) errorf(line, column int, format string, args ...any) error {
	return &Error{
		Path:    s.path,
		Line:    line,
		Column:  column,
		Message: fmt.Sprintf(format, args...),
	}
}

func isIdentifierStart(r rune) bool {
	return r == '_' || r == '$' || r == '@' || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isSymbolRune(r rune) bool {
	switch r {
	case '(', ')', ',', ';', '.', '*', '=', '+', '-', '/', '%', '<', '>', '!', '?', ':', '[', ']', '{', '}', '|':
		return true
	}
	return false
}
