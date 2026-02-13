// Package cache provides caching annotations for SQL queries.
package cache

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CacheConfig holds cache configuration parsed from SQL comments.
type CacheConfig struct {
	Enabled    bool
	TTL        time.Duration
	KeyPattern string
	Invalidate []string
}

// Annotation represents a parsed cache directive.
type Annotation struct {
	TTL        time.Duration
	KeyPattern string
	Invalidate []string
}

var (
	// cacheRegex matches @cache annotations like:
	// @cache ttl=5m key=user:{id}
	// @cache ttl=1h invalidate=users
	// @cache ttl=30s key=post:{slug} invalidate=posts,comments
	cacheRegex = regexp.MustCompile(`^@cache\s+(.+)$`)

	// ttlRegex matches ttl values
	ttlRegex = regexp.MustCompile(`ttl=(\d+)([smhd])`)

	// keyRegex matches key patterns
	keyRegex = regexp.MustCompile(`key=([^\s]+)`)

	// invalidateRegex matches invalidate patterns
	invalidateRegex = regexp.MustCompile(`invalidate=([^\s]+)`)
)

// ParseAnnotation parses a cache annotation from a comment line.
// Returns nil if the line is not a cache annotation.
func ParseAnnotation(line string) *Annotation {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "@cache") {
		return nil
	}

	// Extract the content after @cache
	content := strings.TrimSpace(strings.TrimPrefix(trimmed, "@cache"))
	if content == "" {
		// Bare @cache means enabled with defaults
		return &Annotation{
			TTL: 5 * time.Minute,
		}
	}

	ann := &Annotation{
		TTL: 5 * time.Minute, // default
	}

	// Parse TTL
	if matches := ttlRegex.FindStringSubmatch(content); len(matches) == 3 {
		duration, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		switch unit {
		case "s":
			ann.TTL = time.Duration(duration) * time.Second
		case "m":
			ann.TTL = time.Duration(duration) * time.Minute
		case "h":
			ann.TTL = time.Duration(duration) * time.Hour
		case "d":
			ann.TTL = time.Duration(duration) * 24 * time.Hour
		}
	}

	// Parse key pattern
	if matches := keyRegex.FindStringSubmatch(content); len(matches) == 2 {
		ann.KeyPattern = matches[1]
	}

	// Parse invalidate patterns
	if matches := invalidateRegex.FindStringSubmatch(content); len(matches) == 2 {
		patterns := strings.Split(matches[1], ",")
		ann.Invalidate = make([]string, 0, len(patterns))
		for _, p := range patterns {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				ann.Invalidate = append(ann.Invalidate, trimmed)
			}
		}
	}

	return ann
}

// ToConfig converts an annotation to a full cache config.
func (a *Annotation) ToConfig() CacheConfig {
	if a == nil {
		return CacheConfig{Enabled: false}
	}
	return CacheConfig{
		Enabled:    true,
		TTL:        a.TTL,
		KeyPattern: a.KeyPattern,
		Invalidate: a.Invalidate,
	}
}

// BuildKey builds a cache key from the pattern and parameter values.
// Pattern can contain placeholders like {id}, {slug}, etc.
// params is a map of parameter name to value.
func BuildKey(pattern string, params map[string]string) string {
	if pattern == "" {
		// Generate key from all params
		var parts []string
		for k, v := range params {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(parts, ":")
	}

	// Replace placeholders in pattern
	result := pattern
	for name, value := range params {
		placeholder := fmt.Sprintf("{%s}", name)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// ParseAnnotations extracts cache annotations from comment lines.
func ParseAnnotations(lines []string) *Annotation {
	for _, line := range lines {
		if ann := ParseAnnotation(line); ann != nil {
			return ann
		}
	}
	return nil
}
