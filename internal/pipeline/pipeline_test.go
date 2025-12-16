package pipeline

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/query/block"
)

type memoryWriter struct {
	writes map[string][]byte
	count  int
}

func (w *memoryWriter) WriteFile(path string, data []byte) error {
	if w.writes == nil {
		w.writes = make(map[string][]byte)
	}
	w.count++
	w.writes[path] = append([]byte(nil), data...)
	return nil
}

func TestPipelineDryRun(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
	if len(summary.Files) == 0 {
		t.Fatalf("Files = %v, want generated files", summary.Files)
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times during dry-run, want 0", writer.count)
	}
	if len(summary.Analyses) != 2 {
		t.Fatalf("Analyses = %d, want 2", len(summary.Analyses))
	}

	var (
		listUsersFound bool
		summarizeFound bool
		helperContent  string
		queryFound     bool
	)

	for _, analysis := range summary.Analyses {
		switch analysis.Query.Block.Name {
		case "ListUsers":
			listUsersFound = true
			if analysis.Query.Block.Command != block.CommandMany {
				t.Fatalf("ListUsers command = %v, want CommandMany", analysis.Query.Block.Command)
			}
		case "SummarizeCredits":
			summarizeFound = true
			if analysis.Query.Block.Command != block.CommandOne {
				t.Fatalf("SummarizeCredits command = %v, want CommandOne", analysis.Query.Block.Command)
			}
			if len(analysis.Columns) != 3 {
				t.Fatalf("SummarizeCredits columns = %d, want 3", len(analysis.Columns))
			}
			if col := analysis.Columns[0]; col.Name != "total_users" || col.GoType != "int64" || col.Nullable {
				t.Fatalf("total_users column = %+v, want int64 non-null", col)
			}
			if col := analysis.Columns[1]; col.Name != "sum_credits" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("sum_credits column = %+v, want float64 nullable", col)
			}
			if col := analysis.Columns[2]; col.Name != "avg_credit" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("avg_credit column = %+v, want float64 nullable", col)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("SummarizeCredits params = %v, want none", analysis.Params)
			}
		default:
			t.Fatalf("unexpected query %q in analyses", analysis.Query.Block.Name)
		}
	}
	if !listUsersFound || !summarizeFound {
		t.Fatalf("expected analyses for ListUsers and SummarizeCredits, got %+v", summary.Analyses)
	}

	outPrefix := filepath.Join(filepath.Dir(configPath), "gen") + string(os.PathSeparator)
	for _, file := range summary.Files {
		if !strings.HasPrefix(file.Path, outPrefix) {
			t.Fatalf("file path %q does not reside under %q", file.Path, outPrefix)
		}
		if strings.HasSuffix(file.Path, "_helpers.gen.go") {
			helperContent = string(file.Content)
		}
		if strings.HasSuffix(file.Path, "query_summarize_credits.go") {
			queryFound = true
		}
	}
	if !queryFound {
		t.Fatalf("query_summarize_credits.go not emitted; files = %+v", summary.Files)
	}
	if !strings.Contains(helperContent, "type SummarizeCreditsRow struct") ||
		!strings.Contains(helperContent, "TotalUsers int32") ||
		!strings.Contains(helperContent, "SumCredits sql.NullFloat64") ||
		!strings.Contains(helperContent, "AvgCredit  sql.NullFloat64") && !strings.Contains(helperContent, "AvgCredit sql.NullFloat64") {
		t.Fatalf("_helpers.gen.go missing expected SummarizeCreditsRow fields")
	}
}

func TestPipelineListQueries(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, ListQueries: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Files) != 0 {
		t.Fatalf("Files = %d, want none when listing", len(summary.Files))
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times when listing, want 0", writer.count)
	}
	if len(summary.Analyses) != 2 {
		t.Fatalf("Analyses = %d, want 2", len(summary.Analyses))
	}

	var (
		listUsersFound bool
		summarizeFound bool
	)

	for _, analysis := range summary.Analyses {
		switch analysis.Query.Block.Name {
		case "ListUsers":
			listUsersFound = true
			if analysis.Query.Block.Command != block.CommandMany {
				t.Fatalf("ListUsers command = %v, want CommandMany", analysis.Query.Block.Command)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("ListUsers params = %v, want none", analysis.Params)
			}
		case "SummarizeCredits":
			summarizeFound = true
			if analysis.Query.Block.Command != block.CommandOne {
				t.Fatalf("SummarizeCredits command = %v, want CommandOne", analysis.Query.Block.Command)
			}
			if len(analysis.Columns) != 3 {
				t.Fatalf("SummarizeCredits columns = %d, want 3", len(analysis.Columns))
			}
			if col := analysis.Columns[0]; col.Name != "total_users" || col.GoType != "int64" || col.Nullable {
				t.Fatalf("total_users column = %+v, want int64 non-null", col)
			}
			if col := analysis.Columns[1]; col.Name != "sum_credits" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("sum_credits column = %+v, want float64 nullable", col)
			}
			if col := analysis.Columns[2]; col.Name != "avg_credit" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("avg_credit column = %+v, want float64 nullable", col)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("SummarizeCredits params = %v, want none", analysis.Params)
			}
		default:
			t.Fatalf("unexpected query %q in analyses", analysis.Query.Block.Name)
		}
	}
	if !listUsersFound || !summarizeFound {
		t.Fatalf("expected analyses for ListUsers and SummarizeCredits, got %+v", summary.Analyses)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
}

func prepareFixtures(t *testing.T) string {
	t.Helper()
	src := "testdata"
	dst := t.TempDir()
	copyFile(t, filepath.Join(dst, "config.toml"), filepath.Join(src, "config.toml"))
	return filepath.Join(dst, "config.toml")
}

func copyFile(t *testing.T, dst, src string) {
	t.Helper()
	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		t.Fatalf("open %q: %v", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create %q: %v", dst, err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %q -> %q: %v", src, dst, err)
	}
}
