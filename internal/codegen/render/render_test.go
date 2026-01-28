package render

import (
	"testing"

	goast "go/ast"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name    string
		specs   []Spec
		wantErr bool
	}{
		{
			name: "valid go code",
			specs: []Spec{
				{
					Path: "test.go",
					Raw:  []byte("package main\n\nfunc main() {}"),
				},
			},
			wantErr: false,
		},
		{
			name: "multiple files",
			specs: []Spec{
				{Path: "a.go", Raw: []byte("package main")},
				{Path: "b.go", Raw: []byte("package main")},
			},
			wantErr: false,
		},
		{
			name: "with AST node",
			specs: []Spec{
				{
					Path: "ast.go",
					Node: &goast.File{
						Name: goast.NewIdent("test"),
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil node and raw",
			specs:   []Spec{{Path: "empty.go"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format(tt.specs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.specs) {
				t.Errorf("Format() returned %d files, want %d", len(got), len(tt.specs))
			}
		})
	}
}

func TestRenderedFile(t *testing.T) {
	rf := File{
		Path:    "test.go",
		Content: []byte("package main"),
	}

	if rf.Path != "test.go" {
		t.Errorf("Path = %q, want %q", rf.Path, "test.go")
	}

	if string(rf.Content) != "package main" {
		t.Errorf("Content = %q, want %q", string(rf.Content), "package main")
	}
}

func TestSpec(t *testing.T) {
	node := &goast.File{
		Name: goast.NewIdent("testpkg"),
	}

	spec := Spec{
		Path: "test.go",
		Node: node,
		Raw:  []byte("package testpkg"),
	}

	if spec.Path != "test.go" {
		t.Errorf("Path = %q, want %q", spec.Path, "test.go")
	}
	if spec.Node == nil {
		t.Error("Node should not be nil")
	}
	if string(spec.Raw) != "package testpkg" {
		t.Errorf("Raw = %q, want %q", string(spec.Raw), "package testpkg")
	}
}
