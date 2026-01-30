package fileset

import (
	"fmt"
	"strings"
	"sync"
)

// MemoryResolver implements Resolver for testing without filesystem.
type MemoryResolver struct {
	mu    sync.RWMutex
	files map[string][]byte // path -> content
	base  string
}

// NewMemoryResolver creates a new MemoryResolver with the given files.
// Files should be a map of relative paths to content.
func NewMemoryResolver(base string, files map[string][]byte) *MemoryResolver {
	return &MemoryResolver{
		files: files,
		base:  base,
	}
}

// Resolve matches patterns against in-memory file paths.
func (m *MemoryResolver) Resolve(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, ErrNoPatterns
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matched := false

		// Handle exact paths
		if _, ok := m.files[pattern]; ok {
			if !seen[pattern] {
				results = append(results, pattern)
				seen[pattern] = true
			}
			//nolint:wastedassign // matched is used to skip glob matching below
			matched = true
			continue
		}

		// Handle glob patterns
		for path := range m.files {
			if m.matchPattern(pattern, path) {
				if !seen[path] {
					results = append(results, path)
					seen[path] = true
				}
				matched = true
			}
		}

		if !matched {
			return nil, NoMatchError{Patterns: []string{pattern}}
		}
	}

	return results, nil
}

// matchPattern checks if path matches pattern (simplified glob).
func (m *MemoryResolver) matchPattern(pattern, path string) bool {
	// Handle **/*.sql pattern - matches any .sql file recursively
	if pattern == "**/*.sql" {
		return strings.HasSuffix(path, ".sql")
	}

	// Handle *.sql pattern - matches only root-level .sql files
	if pattern == "*.sql" {
		return strings.HasSuffix(path, ".sql") && !strings.Contains(path, "/")
	}

	// Handle other *.<ext> patterns
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:] // .sql
		return strings.HasSuffix(path, ext) && !strings.Contains(path, "/")
	}

	// Handle exact match
	return pattern == path
}

// ReadFile returns the content of an in-memory file.
func (m *MemoryResolver) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// AddFile adds a file to the resolver.
func (m *MemoryResolver) AddFile(path string, content []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.files == nil {
		m.files = make(map[string][]byte)
	}
	m.files[path] = content
}

// RemoveFile removes a file from the resolver.
func (m *MemoryResolver) RemoveFile(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.files, path)
}

// FileCount returns the number of files.
func (m *MemoryResolver) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.files)
}

// Note: MemoryResolver provides a similar API to Resolver but is not a drop-in
// replacement since Resolver is a concrete struct. It's designed for testing
// scenarios where filesystem I/O needs to be mocked.
