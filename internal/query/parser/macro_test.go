package parser

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/query/block"
)

func TestParseSQLCMacros(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    []Param
		wantErr bool
	}{
		{
			name: "sqlc.slice in IN clause",
			sql:  "DELETE FROM users WHERE id IN (sqlc.slice('ids'))",
			want: []Param{
				{Name: "ids", Style: ParamStyleNamed, IsVariadic: true},
			},
		},
		{
			name: "sqlc.arg named parameter",
			sql:  "UPDATE users SET email = sqlc.arg('email') WHERE id = 1",
			want: []Param{
				{Name: "email", Style: ParamStyleNamed},
			},
		},
		{
			name: "sqlc.narg nullable named parameter",
			sql:  "UPDATE users SET email = sqlc.narg('email') WHERE id = 1",
			want: []Param{
				{Name: "email", Style: ParamStyleNamed}, // Nullability handled in analyzer, parser sees name
			},
		},
		{
			name: "mixed macros",
			sql:  "UPDATE users SET status = sqlc.arg('status') WHERE id IN (sqlc.slice('ids'))",
			want: []Param{
				{Name: "status", Style: ParamStyleNamed},
				{Name: "ids", Style: ParamStyleNamed, IsVariadic: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blk := block.Block{
				Name:    "TestQuery",
				Command: block.CommandMany,
				SQL:     tt.sql,
			}
			got, diags := Parse(blk)

			hasError := false
			for _, d := range diags {
				if d.Severity == SeverityError {
					hasError = true
					break
				}
			}

			if tt.wantErr {
				if !hasError {
					t.Errorf("Parse() expected error, got none")
				}
				return
			}

			if hasError {
				t.Fatalf("Parse() unexpected errors: %v", diags)
			}

			if len(got.Params) != len(tt.want) {
				t.Fatalf("Params length mismatch: got %d, want %d", len(got.Params), len(tt.want))
			}

			for i, wantParam := range tt.want {
				gotParam := got.Params[i]
				if gotParam.Name != wantParam.Name {
					t.Errorf("Param[%d].Name = %q, want %q", i, gotParam.Name, wantParam.Name)
				}
				if gotParam.Style != wantParam.Style {
					t.Errorf("Param[%d].Style = %v, want %v", i, gotParam.Style, wantParam.Style)
				}
				if gotParam.IsVariadic != wantParam.IsVariadic {
					t.Errorf("Param[%d].IsVariadic = %v, want %v", i, gotParam.IsVariadic, wantParam.IsVariadic)
				}
			}
		})
	}
}
