package sqlfix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"log/slog"

	"github.com/electwix/db-catalyst/internal/query/block"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// NewRunner creates a new sqlfix Runner.
func NewRunner() *Runner {
	return &Runner{
		readFile:  os.ReadFile,
		writeFile: defaultWriteFile,
	}
}

// SetCatalog sets the schema catalog used for star expansion and validation.
func (r *Runner) SetCatalog(catalog *model.Catalog, warnings []string) {
	if r == nil {
		return
	}
	r.catalog = catalog
	if len(warnings) == 0 {
		r.catalogWarnings = nil
		return
	}
	r.catalogWarnings = append(r.catalogWarnings[:0], warnings...)
}

// CatalogWarnings returns any warnings encountered during catalog loading.
func (r *Runner) CatalogWarnings() []string {
	if r == nil || len(r.catalogWarnings) == 0 {
		return nil
	}
	out := make([]string, len(r.catalogWarnings))
	copy(out, r.catalogWarnings)
	return out
}

// Runner executes the sqlfix logic on a set of files.
type Runner struct {
	Logger *slog.Logger
	DryRun bool

	catalog         *model.Catalog
	catalogWarnings []string

	readFile  func(string) ([]byte, error)
	writeFile func(string, []byte) error
}

type edit struct {
	start int
	end   int
	text  string
}

// Rewrite processes the given file paths and applies necessary fixes.
func (r *Runner) Rewrite(ctx context.Context, paths []string) ([]Report, error) {
	if r == nil {
		return nil, errors.New("sqlfix: runner is nil")
	}
	if r.readFile == nil {
		r.readFile = os.ReadFile
	}
	if r.writeFile == nil {
		r.writeFile = defaultWriteFile
	}

	reports := make([]Report, 0, len(paths))
	var errs []error
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return reports, err
		}

		report, content, err := r.rewriteFile(ctx, path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		reports = append(reports, report)

		if report.Changed() && !r.DryRun {
			if err := r.writeFile(path, content); err != nil {
				errs = append(errs, fmt.Errorf("write %s: %w", path, err))
			}
		}
	}

	return reports, errors.Join(errs...)
}

func (r *Runner) rewriteFile(ctx context.Context, path string) (Report, []byte, error) {
	report := Report{Path: path}

	data, err := r.readFile(path)
	if err != nil {
		return report, nil, fmt.Errorf("read %s: %w", path, err)
	}

	blocks, err := block.Slice(path, data)
	if err != nil {
		return report, nil, fmt.Errorf("slice %s: %w", path, err)
	}

	edits := make([]edit, 0)
	for _, blk := range blocks {
		if err := ctx.Err(); err != nil {
			return report, nil, err
		}

		currentSQL := blk.SQL
		parseBlock := blk
		parseBlock.SQL = currentSQL

		query, diags := queryparser.Parse(parseBlock)
		recoverable := recoverableColumnErrors(diags)
		if hasParseErrors(diags) && !recoverable {
			report.Warnings = append(report.Warnings, fmt.Sprintf("skip %s:%s due to parse errors", path, blk.Name))
			continue
		}
		if !recoverable && query.Verb != queryparser.VerbSelect {
			continue
		}

		blockChanged := false

		if r.catalog != nil {
			changed, newSQL, newQuery, skip, err := r.processStarExpansion(parseBlock, currentSQL, query, &report, path, blk.Name)
			if err != nil {
				return report, nil, err
			}
			if skip {
				continue
			}
			if changed {
				blockChanged = true
				currentSQL = newSQL
				parseBlock.SQL = currentSQL
				query = newQuery
				recoverable = recoverableColumnErrors(diags)
			}
		}

		if recoverable {
			report.Warnings = append(report.Warnings, fmt.Sprintf("skip %s:%s due to parse errors", path, blk.Name))
			continue
		}

		aliaser := NewAliasGenerator()
		for _, col := range query.Columns {
			if col.Alias != "" {
				aliaser.Reserve(col.Alias)
			}
		}

		aliasEdits := make([]edit, 0)
		for idx, col := range query.Columns {
			if col.Alias != "" {
				continue
			}
			alias := aliaser.Next(col.Expr)
			if alias == "" {
				report.Skipped = append(report.Skipped, SkippedAlias{
					QueryName: blk.Name,
					Expr:      col.Expr,
					Reason:    "unable to derive alias",
				})
				continue
			}
			report.Added = append(report.Added, AddedAlias{
				QueryName:   blk.Name,
				ColumnIndex: idx,
				Alias:       alias,
			})
			aliasEdits = append(aliasEdits, edit{
				start: col.EndOffset,
				end:   col.EndOffset,
				text:  " AS " + alias,
			})
			if r.Logger != nil {
				r.Logger.Debug("added alias", slog.String("file", path), slog.String("query", blk.Name), slog.String("alias", alias))
			}
		}

		if len(aliasEdits) > 0 {
			updatedSQL, err := applyStringEdits(currentSQL, aliasEdits)
			if err != nil {
				return report, nil, fmt.Errorf("apply alias edits for %s:%s: %w", path, blk.Name, err)
			}
			currentSQL = updatedSQL
			blockChanged = true
		}

		if !blockChanged {
			continue
		}

		start := blk.StartOffset
		end := blk.StartOffset + len(blk.SQL)
		edits = append(edits, edit{
			start: start,
			end:   end,
			text:  currentSQL,
		})
	}

	if len(edits) == 0 {
		return report, data, nil
	}

	sort.SliceStable(edits, func(i, j int) bool {
		return edits[i].start < edits[j].start
	})

	updated, err := applyByteEdits(data, edits)
	if err != nil {
		return report, data, fmt.Errorf("apply edits for %s: %w", path, err)
	}

	return report, updated, nil
}

func hasParseErrors(diags []queryparser.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == queryparser.SeverityError {
			return true
		}
	}
	return false
}

func recoverableColumnErrors(diags []queryparser.Diagnostic) bool {
	recoverable := false
	for _, d := range diags {
		if d.Severity != queryparser.SeverityError {
			continue
		}
		recoverable = true
		if !strings.Contains(d.Message, "result column requires alias") {
			return false
		}
	}
	return recoverable
}

func (r *Runner) processStarExpansion(
	parseBlock block.Block,
	currentSQL string,
	query queryparser.Query,
	report *Report,
	path, blockName string,
) (changed bool, newSQL string, newQuery queryparser.Query, skip bool, err error) {
	expandedSQL, warnings, replaced, err := r.expandStars(parseBlock, currentSQL, query)
	if err != nil {
		return false, "", queryparser.Query{}, false, fmt.Errorf("expand stars for %s:%s: %w", path, blockName, err)
	}
	if len(warnings) > 0 {
		report.Warnings = append(report.Warnings, warnings...)
	}
	if replaced == 0 {
		return false, currentSQL, query, false, nil
	}

	parseBlock.SQL = expandedSQL
	newQuery, diags := queryparser.Parse(parseBlock)
	if hasParseErrors(diags) {
		report.Warnings = append(report.Warnings, fmt.Sprintf("skip %s:%s after star expansion due to parse errors", path, blockName))
		return false, "", queryparser.Query{}, true, nil
	}
	if newQuery.Verb != queryparser.VerbSelect {
		return false, "", queryparser.Query{}, true, nil
	}

	report.ExpandedStars += replaced
	return true, expandedSQL, newQuery, false, nil
}

func applyByteEdits(src []byte, edits []edit) ([]byte, error) {
	if len(edits) == 0 {
		return append([]byte(nil), src...), nil
	}

	var buf bytes.Buffer
	cursor := 0
	for _, e := range edits {
		if e.start < 0 || e.start > len(src) || e.end < e.start || e.end > len(src) {
			return nil, fmt.Errorf("invalid edit range [%d,%d)", e.start, e.end)
		}
		if e.start < cursor {
			return nil, fmt.Errorf("overlapping edit starting at %d", e.start)
		}
		buf.Write(src[cursor:e.start])
		buf.WriteString(e.text)
		cursor = e.end
	}
	buf.Write(src[cursor:])
	return buf.Bytes(), nil
}

func defaultWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
