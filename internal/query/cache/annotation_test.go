package cache

import (
	"strings"
	"testing"
	"time"
)

func TestParseAnnotation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Annotation
		wantNil bool
	}{
		{
			name:  "bare @cache",
			input: "@cache",
			want: &Annotation{
				TTL: 5 * time.Minute,
			},
		},
		{
			name:  "with ttl seconds",
			input: "@cache ttl=30s",
			want: &Annotation{
				TTL: 30 * time.Second,
			},
		},
		{
			name:  "with ttl minutes",
			input: "@cache ttl=5m",
			want: &Annotation{
				TTL: 5 * time.Minute,
			},
		},
		{
			name:  "with ttl hours",
			input: "@cache ttl=1h",
			want: &Annotation{
				TTL: 1 * time.Hour,
			},
		},
		{
			name:  "with ttl days",
			input: "@cache ttl=7d",
			want: &Annotation{
				TTL: 7 * 24 * time.Hour,
			},
		},
		{
			name:  "with key pattern",
			input: "@cache ttl=5m key=user:{id}",
			want: &Annotation{
				TTL:        5 * time.Minute,
				KeyPattern: "user:{id}",
			},
		},
		{
			name:  "with invalidate",
			input: "@cache ttl=5m invalidate=users",
			want: &Annotation{
				TTL:        5 * time.Minute,
				Invalidate: []string{"users"},
			},
		},
		{
			name:  "with multiple invalidate patterns",
			input: "@cache ttl=5m invalidate=users,posts,comments",
			want: &Annotation{
				TTL:        5 * time.Minute,
				Invalidate: []string{"users", "posts", "comments"},
			},
		},
		{
			name:  "full annotation",
			input: "@cache ttl=5m key=user:{id} invalidate=users,posts",
			want: &Annotation{
				TTL:        5 * time.Minute,
				KeyPattern: "user:{id}",
				Invalidate: []string{"users", "posts"},
			},
		},
		{
			name:    "not a cache annotation",
			input:   "@param id: uuid",
			wantNil: true,
		},
		{
			name:    "regular comment",
			input:   "Get user by ID",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAnnotation(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseAnnotation() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseAnnotation() = nil, want %v", tt.want)
			}
			if got.TTL != tt.want.TTL {
				t.Errorf("TTL = %v, want %v", got.TTL, tt.want.TTL)
			}
			if got.KeyPattern != tt.want.KeyPattern {
				t.Errorf("KeyPattern = %q, want %q", got.KeyPattern, tt.want.KeyPattern)
			}
			if len(got.Invalidate) != len(tt.want.Invalidate) {
				t.Errorf("Invalidate length = %d, want %d", len(got.Invalidate), len(tt.want.Invalidate))
			} else {
				for i := range got.Invalidate {
					if got.Invalidate[i] != tt.want.Invalidate[i] {
						t.Errorf("Invalidate[%d] = %q, want %q", i, got.Invalidate[i], tt.want.Invalidate[i])
					}
				}
			}
		})
	}
}

func TestBuildKey(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		params  map[string]string
		want    string
	}{
		{
			name:    "empty pattern",
			pattern: "",
			params:  map[string]string{"id": "123", "slug": "hello"},
			want:    "id=123:slug=hello", // order may vary
		},
		{
			name:    "simple pattern",
			pattern: "user:{id}",
			params:  map[string]string{"id": "123"},
			want:    "user:123",
		},
		{
			name:    "multiple placeholders",
			pattern: "post:{slug}:comment:{id}",
			params:  map[string]string{"slug": "hello", "id": "456"},
			want:    "post:hello:comment:456",
		},
		{
			name:    "pattern with extra params",
			pattern: "user:{id}",
			params:  map[string]string{"id": "123", "unused": "value"},
			want:    "user:123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildKey(tt.pattern, tt.params)
			// For empty pattern, we just check it contains expected parts
			if tt.pattern == "" {
				if !containsAll(got, []string{"id=123", "slug=hello"}) {
					t.Errorf("BuildKey() = %q, want to contain id=123 and slug=hello", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("BuildKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}

func TestParseAnnotations(t *testing.T) {
	lines := []string{
		"Get user by ID",
		"@cache ttl=5m key=user:{id}",
		"Some other comment",
	}

	got := ParseAnnotations(lines)
	if got == nil {
		t.Fatal("ParseAnnotations() = nil, want annotation")
	}
	if got.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want 5m", got.TTL)
	}
	if got.KeyPattern != "user:{id}" {
		t.Errorf("KeyPattern = %q, want user:{id}", got.KeyPattern)
	}
}

func TestToConfig(t *testing.T) {
	tests := []struct {
		name string
		ann  *Annotation
		want CacheConfig
	}{
		{
			name: "nil annotation",
			ann:  nil,
			want: CacheConfig{Enabled: false},
		},
		{
			name: "enabled annotation",
			ann: &Annotation{
				TTL:        5 * time.Minute,
				KeyPattern: "user:{id}",
				Invalidate: []string{"users"},
			},
			want: CacheConfig{
				Enabled:    true,
				TTL:        5 * time.Minute,
				KeyPattern: "user:{id}",
				Invalidate: []string{"users"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ann.ToConfig()
			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.want.Enabled)
			}
			if got.TTL != tt.want.TTL {
				t.Errorf("TTL = %v, want %v", got.TTL, tt.want.TTL)
			}
			if got.KeyPattern != tt.want.KeyPattern {
				t.Errorf("KeyPattern = %q, want %q", got.KeyPattern, tt.want.KeyPattern)
			}
			if len(got.Invalidate) != len(tt.want.Invalidate) {
				t.Errorf("Invalidate length = %d, want %d", len(got.Invalidate), len(tt.want.Invalidate))
			}
		})
	}
}
