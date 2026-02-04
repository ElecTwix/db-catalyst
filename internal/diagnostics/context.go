// Package diagnostics provides rich diagnostic information for db-catalyst.
package diagnostics

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// ContextExtractor extracts code context from source files.
type ContextExtractor struct {
	// Cache of file contents to avoid re-reading files
	cache map[string][]string
}

// NewContextExtractor creates a new context extractor.
func NewContextExtractor() *ContextExtractor {
	return &ContextExtractor{
		cache: make(map[string][]string),
	}
}

// ExtractContext extracts lines around a specific location.
// Returns the context lines and the position within those lines.
func (e *ContextExtractor) ExtractContext(path string, line, column int, contextLines int) (Context, error) {
	lines, err := e.getLines(path)
	if err != nil {
		return Context{}, err
	}

	if line < 1 || line > len(lines) {
		return Context{}, fmt.Errorf("line %d out of range [1, %d]", line, len(lines))
	}

	// Calculate the range of lines to include
	startLine := line - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := line + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Extract the context lines
	contextLines_ := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		contextLines_ = append(contextLines_, lines[i-1]) // lines is 0-indexed
	}

	return Context{
		Lines:       contextLines_,
		StartLine:   startLine,
		ErrorLine:   line,
		ErrorColumn: column,
	}, nil
}

// ExtractSpan extracts context for a span of lines.
func (e *ContextExtractor) ExtractSpan(path string, startLine, endLine, startCol, endCol int) (Context, error) {
	lines, err := e.getLines(path)
	if err != nil {
		return Context{}, err
	}

	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	contextLines_ := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		contextLines_ = append(contextLines_, lines[i-1])
	}

	return Context{
		Lines:       contextLines_,
		StartLine:   startLine,
		ErrorLine:   startLine,
		ErrorColumn: startCol,
		EndLine:     endLine,
		EndColumn:   endCol,
		IsSpan:      true,
	}, nil
}

func (e *ContextExtractor) getLines(path string) ([]string, error) {
	// Check cache first
	if lines, ok := e.cache[path]; ok {
		return lines, nil
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	// Split into lines
	lines := splitLines(content)

	// Cache for future use
	e.cache[path] = lines

	return lines, nil
}

func splitLines(data []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// Context represents extracted code context.
type Context struct {
	Lines       []string
	StartLine   int
	ErrorLine   int
	ErrorColumn int
	EndLine     int
	EndColumn   int
	IsSpan      bool
}

// IsEmpty returns true if the context has no lines.
func (c Context) IsEmpty() bool {
	return len(c.Lines) == 0
}

// Format formats the context for display with line numbers.
func (c Context) Format() string {
	if c.IsEmpty() {
		return ""
	}

	var b strings.Builder
	maxLineNum := c.StartLine + len(c.Lines) - 1
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range c.Lines {
		lineNum := c.StartLine + i
		isErrorLine := lineNum == c.ErrorLine

		// Line number
		if isErrorLine {
			fmt.Fprintf(&b, "> %*d | ", lineNumWidth, lineNum)
		} else {
			fmt.Fprintf(&b, "  %*d | ", lineNumWidth, lineNum)
		}

		// Line content
		b.WriteString(line)
		b.WriteString("\n")

		// Error indicator
		if isErrorLine && c.ErrorColumn > 0 {
			// Spaces for line number and padding
			for j := 0; j < lineNumWidth+4; j++ {
				b.WriteByte(' ')
			}
			// Spaces up to error column
			for j := 0; j < c.ErrorColumn-1 && j < len(line); j++ {
				if line[j] == '\t' {
					b.WriteByte('\t')
				} else {
					b.WriteByte(' ')
				}
			}
			b.WriteString("^\n")
		}
	}

	return b.String()
}

// String returns a simple string representation of the context.
func (c Context) String() string {
	return strings.Join(c.Lines, "\n")
}

// GetErrorLine returns the line containing the error.
func (c Context) GetErrorLine() string {
	if c.IsEmpty() {
		return ""
	}
	idx := c.ErrorLine - c.StartLine
	if idx >= 0 && idx < len(c.Lines) {
		return c.Lines[idx]
	}
	return ""
}

// SnippetExtractor extracts short code snippets from SQL/content.
type SnippetExtractor struct {
	maxLength int
}

// NewSnippetExtractor creates a new snippet extractor.
func NewSnippetExtractor() *SnippetExtractor {
	return &SnippetExtractor{maxLength: 80}
}

// WithMaxLength sets the maximum snippet length.
func (e *SnippetExtractor) WithMaxLength(maxLen int) *SnippetExtractor {
	e.maxLength = maxLen
	return e
}

// Extract extracts a snippet around a position in content.
func (e *SnippetExtractor) Extract(content []byte, offset int) string {
	if offset < 0 || offset >= len(content) {
		return ""
	}

	// Find the start of the snippet
	start := offset - e.maxLength/2
	if start < 0 {
		start = 0
	}

	// Find the end of the snippet
	end := offset + e.maxLength/2
	if end > len(content) {
		end = len(content)
	}

	// Adjust to word boundaries if possible
	start = e.findWordBoundary(content, start, true)
	end = e.findWordBoundary(content, end, false)

	snippet := string(content[start:end])

	// Add ellipsis if truncated
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

func (e *SnippetExtractor) findWordBoundary(content []byte, pos int, forward bool) int {
	if forward {
		// Move forward to find the start of a word
		for pos < len(content) && isWordChar(content[pos]) {
			pos++
		}
		for pos < len(content) && !isWordChar(content[pos]) {
			pos++
		}
	} else {
		// Move backward to find the end of a word
		for pos > 0 && isWordChar(content[pos-1]) {
			pos--
		}
		for pos > 0 && !isWordChar(content[pos-1]) {
			pos--
		}
	}
	return pos
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// ExtractLine extracts a specific line from content.
func ExtractLine(content []byte, lineNum int) (string, error) {
	lines := splitLines(content)
	if lineNum < 1 || lineNum > len(lines) {
		return "", fmt.Errorf("line %d out of range [1, %d]", lineNum, len(lines))
	}
	return lines[lineNum-1], nil
}

// ExtractLines extracts a range of lines from content.
func ExtractLines(content []byte, startLine, endLine int) ([]string, error) {
	lines := splitLines(content)
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return nil, fmt.Errorf("invalid line range: %d-%d", startLine, endLine)
	}

	result := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		result = append(result, lines[i-1])
	}
	return result, nil
}
