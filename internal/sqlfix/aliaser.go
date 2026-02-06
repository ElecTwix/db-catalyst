// Package sqlfix implements tools for automated SQL rewrites and migration.
//
//nolint:goconst // SQL keywords and column names are naturally repeated
package sqlfix

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// AliasGenerator generates unique column aliases for SQL queries.
type AliasGenerator struct {
	counts map[string]int
}

// NewAliasGenerator creates a new AliasGenerator.
func NewAliasGenerator() *AliasGenerator {
	return &AliasGenerator{counts: make(map[string]int)}
}

// Reserve marks an alias as already used to prevent collisions.
func (g *AliasGenerator) Reserve(alias string) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if g.counts == nil {
		g.counts = make(map[string]int)
	}
	sanitized := sanitizeAlias(alias)
	if sanitized == "" {
		return
	}
	if !isValidIdentifierStart(rune(sanitized[0])) {
		sanitized = "col_" + sanitized
	}
	if g.counts[sanitized] < 1 {
		g.counts[sanitized] = 1
	}
}

// Next generates a deterministic, unique alias for the given expression.
func (g *AliasGenerator) Next(expr string) string {
	base := deriveAliasBase(expr)
	if base == "" {
		base = "expr"
	}
	if !isValidIdentifierStart(rune(base[0])) {
		base = "col_" + base
	}
	count := g.counts[base]
	if count == 0 {
		g.counts[base] = 1
		return base
	}
	count++
	g.counts[base] = count
	return base + "_" + strconv.Itoa(count)
}

var (
	aggregatePattern = regexp.MustCompile(`^(?i)([A-Z_]+)\s*\((.*)\)$`)
	numericPattern   = regexp.MustCompile(`^[+-]?\d+(?:\.\d+)?$`)
)

func deriveAliasBase(expr string) string {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return ""
	}

	if m := aggregatePattern.FindStringSubmatch(trimmed); len(m) == 3 {
		fn := strings.ToLower(m[1])
		arg := strings.TrimSpace(m[2])
		if arg == "*" || arg == "1" {
			return sanitizeAlias(fn + "_all")
		}
		last := arg
		if idx := strings.LastIndex(arg, "."); idx >= 0 {
			last = arg[idx+1:]
		}
		last = strings.Trim(last, "`\"")
		if last == "" {
			last = "expr"
		}
		return sanitizeAlias(fn + "_" + last)
	}

	upper := strings.ToUpper(trimmed)
	if upper == "TRUE" || upper == "FALSE" {
		return sanitizeAlias("flag_" + strings.ToLower(upper))
	}

	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2 {
		val := trimmed[1 : len(trimmed)-1]
		val = strings.TrimSpace(val)
		if val == "" {
			val = "str"
		}
		return sanitizeAlias("const_" + val)
	}

	if numericPattern.MatchString(trimmed) {
		compressed := strings.ReplaceAll(trimmed, ".", "_")
		return sanitizeAlias("const_" + compressed)
	}

	parts := splitIdentifierParts(trimmed)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return sanitizeAlias(strings.Join(parts, "_"))
}

func splitIdentifierParts(expr string) []string {
	buf := strings.Builder{}
	parts := make([]string, 0, 4)
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		segment := buf.String()
		buf.Reset()
		segment = strings.Trim(segment, "`")
		if segment == "" {
			return
		}
		parts = append(parts, segment)
	}
	for _, r := range expr {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			buf.WriteRune(unicode.ToLower(r))
		case r == '.':
			flush()
		case unicode.IsSpace(r):
			flush()
		case r == '+' || r == '-' || r == '*':
			flush()
			parts = append(parts, operatorWord(r))
		default:
			flush()
		}
	}
	flush()
	return dedupeParts(parts)
}

func dedupeParts(parts []string) []string {
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if len(result) == 0 || result[len(result)-1] != part {
			result = append(result, part)
		}
	}
	return result
}

func operatorWord(r rune) string {
	switch r {
	case '+':
		return "plus"
	case '-':
		return "minus"
	case '*':
		return "mul"
	default:
		return "op"
	}
}

func sanitizeAlias(alias string) string {
	alias = strings.ToLower(alias)
	var b strings.Builder
	b.Grow(len(alias))
	prevUnderscore := false
	for _, r := range alias {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if r == '_' {
				if prevUnderscore {
					continue
				}
				prevUnderscore = true
				b.WriteRune(r)
				continue
			}
			prevUnderscore = false
			b.WriteRune(r)
			continue
		}
		if !prevUnderscore {
			b.WriteRune('_')
		}
		prevUnderscore = true
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "expr"
	}
	return out
}

func isValidIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
