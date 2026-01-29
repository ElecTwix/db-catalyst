package ast

import (
	"errors"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const goKeywordSuffix = "_"

// ExportedIdentifier converts raw input into a public Go identifier.
func ExportedIdentifier(raw string) string {
	ident := toIdentifier(raw, true)
	if ident == "" {
		ident = "X"
	}
	if token.Lookup(ident).IsKeyword() {
		ident += goKeywordSuffix
	}
	return ident
}

// UnexportedIdentifier converts raw input into a private Go identifier.
func UnexportedIdentifier(raw string) string {
	ident := toIdentifier(raw, false)
	if ident == "" {
		ident = "value"
	}
	if token.Lookup(ident).IsKeyword() {
		ident += goKeywordSuffix
	}
	return ident
}

// FileName converts the raw name into a snake_case file name segment.
func FileName(raw string) string {
	if raw == "" {
		return "query"
	}
	runes := []rune(raw)
	var b strings.Builder
	b.Grow(len(runes) * 2)
	prevUnderscore := false
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.IsUpper(r) {
				if i > 0 && !prevUnderscore {
					b.WriteRune('_')
				}
				r = unicode.ToLower(r)
			}
			b.WriteRune(r)
			prevUnderscore = false
		case r == '_' || r == '-' || r == ' ':
			if !prevUnderscore && b.Len() > 0 {
				b.WriteRune('_')
				prevUnderscore = true
			}
		default:
			// skip unsupported characters
		}
	}
	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "query"
	}
	return name
}

func toIdentifier(raw string, exported bool) string {
	if raw == "" {
		return ""
	}
	segments := splitSegments(raw)
	if len(segments) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		lower := strings.ToLower(seg)
		r, size := utf8.DecodeRuneInString(lower)
		if r == utf8.RuneError {
			continue
		}
		if i == 0 && !exported {
			b.WriteRune(unicode.ToLower(r))
			b.WriteString(lower[size:])
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(lower[size:])
	}
	ident := b.String()
	if ident == "" {
		return ident
	}
	r, _ := utf8.DecodeRuneInString(ident)
	if !unicode.IsLetter(r) && r != '_' {
		if exported {
			ident = "X" + ident
		} else {
			ident = "x" + ident
		}
	}
	return ident
}

func splitSegments(raw string) []string {
	parts := make([]string, 0, 4)
	var buf strings.Builder
	runes := []rune(raw)
	flush := func() {
		if buf.Len() > 0 {
			parts = append(parts, buf.String())
			buf.Reset()
		}
	}
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			prev := rune(0)
			next := rune(0)
			if i > 0 {
				prev = runes[i-1]
			}
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsUpper(r) {
				if i > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
					flush()
				} else if i > 0 && unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next) {
					flush()
				}
			}
			buf.WriteRune(r)
		case r == '_' || r == '-' || r == ' ':
			flush()
		default:
			flush()
		}
	}
	flush()
	return parts
}

// UniqueName ensures returned identifier does not collide with previous values.
func UniqueName(base string, used map[string]int) (string, error) {
	if base == "" {
		base = "value"
	}
	if used == nil {
		return "", errors.New("nil name map")
	}
	if _, exists := used[base]; !exists {
		used[base] = 1
		return base, nil
	}
	for i := used[base] + 1; ; i++ {
		candidate := base + strconv.Itoa(i)
		if _, exists := used[candidate]; !exists {
			used[base] = i
			used[candidate] = 1
			return candidate, nil
		}
	}
}
