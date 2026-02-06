// Package block handles the parsing of SQL query blocks.
package block

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Command represents the type of query (one, many, exec, etc.).
type Command int

const (
	// CommandUnknown indicates an unrecognized command.
	CommandUnknown Command = iota
	// CommandOne indicates a query returning a single row.
	CommandOne
	// CommandMany indicates a query returning multiple rows.
	CommandMany
	// CommandExec indicates a query that executes without returning rows.
	CommandExec
	// CommandExecResult indicates a query that returns execution result (rows affected, etc.).
	CommandExecResult
)

// Block represents a parsed SQL query block.
type Block struct {
	Path        string
	Name        string
	Command     Command
	SQL         string
	Doc         string
	Line        int
	Column      int
	StartOffset int
	EndOffset   int
	Suffix      string
}

func (c Command) String() string {
	switch c {
	case CommandOne:
		return ":one"
	case CommandMany:
		return ":many"
	case CommandExec:
		return ":exec"
	case CommandExecResult:
		return ":execresult"
	default:
		return ":unknown"
	}
}

// ParseCommand parses a command tag (e.g., ":one") into a Command.
func ParseCommand(tag string) (Command, bool) {
	switch strings.ToLower(tag) {
	case ":one":
		return CommandOne, true
	case ":many":
		return CommandMany, true
	case ":exec":
		return CommandExec, true
	case ":execresult":
		return CommandExecResult, true
	default:
		return CommandUnknown, false
	}
}

// Slice extracts query blocks from a SQL file.
func Slice(path string, src []byte) ([]Block, error) {
	if !utf8.Valid(src) {
		return nil, fmt.Errorf("%s:1:1: input is not valid UTF-8", path)
	}
	text := string(src)
	lines := splitLines(text)
	type markerInfo struct {
		name         string
		command      Command
		line         lineInfo
		column       int
		docLines     []string
		docStart     int
		contentStart int
		lineIndex    int
	}
	markers := make([]markerInfo, 0, len(lines))
	for idx, ln := range lines {
		trimmedLeft := strings.TrimLeft(ln.text, "\t ")
		if !strings.HasPrefix(trimmedLeft, "--") {
			continue
		}
		content := strings.TrimSpace(trimmedLeft[2:])
		lowerContent := strings.ToLower(content)
		if !strings.HasPrefix(lowerContent, "name:") {
			continue
		}
		rest := strings.TrimSpace(content[len("name:"):])
		fields := strings.Fields(rest)
		column := len(ln.text) - len(trimmedLeft) + 1
		if len(fields) == 0 {
			return nil, fmt.Errorf("%s:%d:%d: missing block name", path, ln.line, column)
		}
		if len(fields) < 2 { //nolint:mnd // need exactly 2 fields: name and command
			return nil, fmt.Errorf("%s:%d:%d: missing command for block %q", path, ln.line, column, fields[0])
		}
		if len(fields) > 2 { //nolint:mnd // need exactly 2 fields: name and command
			return nil, fmt.Errorf("%s:%d:%d: unexpected tokens after command for block %q", path, ln.line, column, fields[0])
		}
		name := fields[0]
		if !isIdent(name) {
			return nil, fmt.Errorf("%s:%d:%d: invalid block name %q", path, ln.line, column+len("-- name:"), name)
		}
		cmdTag := fields[1]
		cmd, ok := ParseCommand(cmdTag)
		if !ok {
			return nil, fmt.Errorf("%s:%d:%d: unknown command %s", path, ln.line, column, cmdTag)
		}
		docLines, docStart := collectDocLines(lines, idx)
		markers = append(markers, markerInfo{
			name:         name,
			command:      cmd,
			line:         ln,
			column:       column,
			docLines:     docLines,
			docStart:     docStart,
			contentStart: ln.next,
			lineIndex:    idx,
		})
	}
	if len(markers) == 0 {
		if len(lines) == 0 {
			return nil, nil
		}
		for _, ln := range lines {
			trimmed := strings.TrimSpace(ln.text)
			if trimmed == "" {
				continue
			}
			trimmedLeft := strings.TrimLeft(ln.text, "\t ")
			if strings.HasPrefix(trimmedLeft, "--") {
				continue
			}
			column := len(ln.text) - len(trimmedLeft) + 1
			return nil, fmt.Errorf("%s:%d:%d: encountered SQL before block marker", path, ln.line, column)
		}
		if strings.TrimSpace(text) != "" {
			return nil, fmt.Errorf("%s:1:1: no query blocks found", path)
		}
		return nil, nil
	}
	first := markers[0]
	for i := 0; i < first.lineIndex; i++ {
		trimmed := strings.TrimSpace(lines[i].text)
		if trimmed == "" {
			continue
		}
		trimmedLeft := strings.TrimLeft(lines[i].text, "\t ")
		if strings.HasPrefix(trimmedLeft, "--") {
			continue
		}
		column := len(lines[i].text) - len(trimmedLeft) + 1
		return nil, fmt.Errorf("%s:%d:%d: encountered SQL before block marker", path, lines[i].line, column)
	}
	blocks := make([]Block, 0, len(markers))
	for idx, m := range markers {
		sqlStart := m.contentStart
		sqlEnd := len(text)
		if idx+1 < len(markers) {
			next := markers[idx+1]
			sqlEnd = next.docStart
		}
		if sqlStart > sqlEnd {
			sqlStart = sqlEnd
		}
		raw := text[sqlStart:sqlEnd]
		sql := trimSQL(raw)
		suffix := ""
		if len(sql) < len(raw) {
			suffix = raw[len(sql):]
		}
		blocks = append(blocks, Block{
			Path:        path,
			Name:        m.name,
			Command:     m.command,
			SQL:         sql,
			Doc:         formatDoc(m.docLines),
			Line:        m.line.line,
			Column:      m.column,
			StartOffset: sqlStart,
			EndOffset:   sqlEnd,
			Suffix:      suffix,
		})
	}
	return blocks, nil
}

type lineInfo struct {
	start int
	end   int
	next  int
	text  string
	line  int
}

func splitLines(text string) []lineInfo {
	if len(text) == 0 {
		return nil
	}
	lines := make([]lineInfo, 0, strings.Count(text, "\n")+1)
	idx := 0
	lineNumber := 1
	for idx < len(text) {
		start := idx
		for idx < len(text) && text[idx] != '\n' && text[idx] != '\r' {
			idx++
		}
		end := idx
		next := idx
		if next < len(text) {
			switch text[next] {
			case '\r':
				next++
				if next < len(text) && text[next] == '\n' {
					next++
				}
			case '\n':
				next++
			}
		}
		lines = append(lines, lineInfo{
			start: start,
			end:   end,
			next:  next,
			text:  text[start:end],
			line:  lineNumber,
		})
		idx = next
		lineNumber++
	}
	return lines
}

func collectDocLines(lines []lineInfo, markerIdx int) ([]string, int) {
	if markerIdx == 0 {
		return nil, lines[markerIdx].start
	}
	doc := make([]string, 0)
	docStart := lines[markerIdx].start
	for i := markerIdx - 1; i >= 0; i-- {
		text := lines[i].text
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			break
		}
		trimmedLeft := strings.TrimLeft(text, "\t ")
		if !strings.HasPrefix(trimmedLeft, "--") {
			break
		}
		content := strings.TrimSpace(trimmedLeft[2:])
		lowerContent := strings.ToLower(content)
		if strings.HasPrefix(lowerContent, "name:") {
			break
		}
		doc = append(doc, content)
		docStart = lines[i].start
	}
	if len(doc) == 0 {
		return nil, lines[markerIdx].start
	}
	slices.Reverse(doc)
	return doc, docStart
}

func trimSQL(sql string) string {
	return strings.TrimRightFunc(sql, unicode.IsSpace)
}

func formatDoc(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func isIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
