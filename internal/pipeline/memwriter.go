package pipeline

import (
	"sync"
)

// MemoryWriter implements Writer for testing without filesystem I/O.
type MemoryWriter struct {
	mu    sync.RWMutex
	Files map[string][]byte
}

// WriteFile stores data in memory.
func (m *MemoryWriter) WriteFile(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[path] = data
	return nil
}

// GetFile retrieves a file's content.
func (m *MemoryWriter) GetFile(path string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.Files[path]
	return data, ok
}

// HasFile checks if a file exists.
func (m *MemoryWriter) HasFile(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.Files[path]
	return ok
}

// Clear removes all files.
func (m *MemoryWriter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Files = make(map[string][]byte)
}

// FileCount returns the number of files.
func (m *MemoryWriter) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.Files)
}

// Ensure MemoryWriter implements Writer interface
var _ Writer = (*MemoryWriter)(nil)
