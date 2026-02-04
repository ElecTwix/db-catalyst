package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextExtractor(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	content := "SELECT col1\nFROM users\nWHERE id = ?\nORDER BY name"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	extractor := NewContextExtractor()

	t.Run("ExtractContext", func(t *testing.T) {
		ctx, err := extractor.ExtractContext(tmpFile, 2, 5, 1)
		if err != nil {
			t.Fatalf("ExtractContext failed: %v", err)
		}

		if ctx.StartLine != 1 {
			t.Errorf("StartLine = %d, want 1", ctx.StartLine)
		}
		if ctx.ErrorLine != 2 {
			t.Errorf("ErrorLine = %d, want 2", ctx.ErrorLine)
		}
		if ctx.ErrorColumn != 5 {
			t.Errorf("ErrorColumn = %d, want 5", ctx.ErrorColumn)
		}
		if len(ctx.Lines) != 3 {
			t.Errorf("Lines length = %d, want 3", len(ctx.Lines))
		}
	})

	t.Run("ExtractContextOutOfRange", func(t *testing.T) {
		_, err := extractor.ExtractContext(tmpFile, 10, 1, 1)
		if err == nil {
			t.Error("Expected error for out of range line")
		}
	})

	t.Run("ExtractSpan", func(t *testing.T) {
		ctx, err := extractor.ExtractSpan(tmpFile, 1, 3, 1, 10)
		if err != nil {
			t.Fatalf("ExtractSpan failed: %v", err)
		}

		if !ctx.IsSpan {
			t.Error("IsSpan should be true")
		}
		if ctx.EndLine != 3 {
			t.Errorf("EndLine = %d, want 3", ctx.EndLine)
		}
	})
}

func TestContextExtractorCaching(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	content := "SELECT 1"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	extractor := NewContextExtractor()

	// First extraction should read from disk
	_, err := extractor.ExtractContext(tmpFile, 1, 1, 0)
	if err != nil {
		t.Fatalf("First extraction failed: %v", err)
	}

	// Second extraction should use cache
	_, err = extractor.ExtractContext(tmpFile, 1, 1, 0)
	if err != nil {
		t.Fatalf("Second extraction failed: %v", err)
	}

	// Verify cache has entry
	if _, ok := extractor.cache[tmpFile]; !ok {
		t.Error("Expected file to be cached")
	}
}

func TestContextFormat(t *testing.T) {
	ctx := Context{
		Lines:       []string{"SELECT col1,", "  col2,", "  col3", "FROM users"},
		StartLine:   1,
		ErrorLine:   2,
		ErrorColumn: 3,
	}

	formatted := ctx.Format()

	// Check that error line is marked
	if !strings.Contains(formatted, "> 2 |") {
		t.Errorf("Format should mark error line with >, got:\n%s", formatted)
	}

	// Check that non-error lines are not marked
	if strings.Contains(formatted, "> 1 |") {
		t.Errorf("Format should not mark non-error lines with >, got:\n%s", formatted)
	}

	// Check for error indicator (^)
	if !strings.Contains(formatted, "^") {
		t.Errorf("Format should include error indicator (^), got:\n%s", formatted)
	}
}

func TestContextFormatEmpty(t *testing.T) {
	ctx := Context{}
	formatted := ctx.Format()
	if formatted != "" {
		t.Errorf("Format() = %q, want empty string", formatted)
	}
}

func TestContextIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		ctx      Context
		expected bool
	}{
		{"empty", Context{}, true},
		{"with lines", Context{Lines: []string{"SELECT 1"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ctx.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContextGetErrorLine(t *testing.T) {
	tests := []struct {
		name     string
		ctx      Context
		expected string
	}{
		{
			name:     "normal",
			ctx:      Context{Lines: []string{"line1", "line2", "line3"}, StartLine: 1, ErrorLine: 2},
			expected: "line2",
		},
		{
			name:     "empty",
			ctx:      Context{},
			expected: "",
		},
		{
			name:     "out of range",
			ctx:      Context{Lines: []string{"line1"}, StartLine: 1, ErrorLine: 5},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ctx.GetErrorLine(); got != tt.expected {
				t.Errorf("GetErrorLine() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestContextString(t *testing.T) {
	ctx := Context{
		Lines: []string{"line1", "line2", "line3"},
	}

	got := ctx.String()
	want := "line1\nline2\nline3"

	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestSnippetExtractor(t *testing.T) {
	extractor := NewSnippetExtractor()

	t.Run("Extract", func(t *testing.T) {
		content := []byte("SELECT col1, col2, col3 FROM users WHERE id = 1")
		snippet := extractor.Extract(content, 20)

		if snippet == "" {
			t.Error("Extract returned empty string")
		}

		// Should contain content around offset 20
		if !strings.Contains(snippet, "col2") {
			t.Errorf("Snippet should contain 'col2', got: %s", snippet)
		}
	})

	t.Run("ExtractWithMaxLength", func(t *testing.T) {
		extractor := NewSnippetExtractor().WithMaxLength(20)
		content := []byte("SELECT col1, col2, col3 FROM users WHERE id = 1")
		snippet := extractor.Extract(content, 20)

		if len(snippet) > 30 { // Allow for ellipsis
			t.Errorf("Snippet too long: %d chars", len(snippet))
		}
	})

	t.Run("ExtractOutOfRange", func(t *testing.T) {
		content := []byte("SELECT 1")
		snippet := extractor.Extract(content, 100)

		if snippet != "" {
			t.Errorf("Extract should return empty for out of range, got: %s", snippet)
		}
	})

	t.Run("ExtractNegativeOffset", func(t *testing.T) {
		content := []byte("SELECT 1")
		snippet := extractor.Extract(content, -1)

		if snippet != "" {
			t.Errorf("Extract should return empty for negative offset, got: %s", snippet)
		}
	})
}

func TestExtractLine(t *testing.T) {
	content := []byte("line1\nline2\nline3")

	tests := []struct {
		lineNum int
		want    string
		wantErr bool
	}{
		{1, "line1", false},
		{2, "line2", false},
		{3, "line3", false},
		{0, "", true},
		{4, "", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("line%d", tt.lineNum), func(t *testing.T) {
			got, err := ExtractLine(content, tt.lineNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractLines(t *testing.T) {
	content := []byte("line1\nline2\nline3\nline4")

	tests := []struct {
		name      string
		startLine int
		endLine   int
		want      []string
		wantErr   bool
	}{
		{
			name:      "valid range",
			startLine: 2,
			endLine:   3,
			want:      []string{"line2", "line3"},
			wantErr:   false,
		},
		{
			name:      "single line",
			startLine: 2,
			endLine:   2,
			want:      []string{"line2"},
			wantErr:   false,
		},
		{
			name:      "clamped start",
			startLine: 0,
			endLine:   2,
			want:      []string{"line1", "line2"},
			wantErr:   false,
		},
		{
			name:      "clamped end",
			startLine: 3,
			endLine:   10,
			want:      []string{"line3", "line4"},
			wantErr:   false,
		},
		{
			name:      "invalid range",
			startLine: 3,
			endLine:   2,
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractLines(content, tt.startLine, tt.endLine)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ExtractLines() returned %d lines, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    []string
	}{
		{
			name:    "unix line endings",
			content: []byte("line1\nline2\nline3"),
			want:    []string{"line1", "line2", "line3"},
		},
		{
			name:    "windows line endings",
			content: []byte("line1\r\nline2\r\nline3"),
			want:    []string{"line1", "line2", "line3"},
		},
		{
			name:    "empty",
			content: []byte(""),
			want:    nil,
		},
		{
			name:    "single line",
			content: []byte("only line"),
			want:    []string{"only line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("splitLines() returned %d lines, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		char byte
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'5', true},
		{'_', true},
		{' ', false},
		{'.', false},
		{',', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			if got := isWordChar(tt.char); got != tt.want {
				t.Errorf("isWordChar(%q) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}
