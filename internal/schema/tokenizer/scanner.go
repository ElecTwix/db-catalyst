package tokenizer

import (
	"fmt"
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
	for _, part := range strings.Split(clean, "\n") {
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
