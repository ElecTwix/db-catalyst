package sqlfix

// AddedAlias records an alias added to a query column.
type AddedAlias struct {
	QueryName   string
	ColumnIndex int
	Alias       string
}

// SkippedAlias records a column that could not be aliased.
type SkippedAlias struct {
	QueryName string
	Expr      string
	Reason    string
}

// Report contains the results of a sqlfix run on a single file.
type Report struct {
	Path          string
	Added         []AddedAlias
	Skipped       []SkippedAlias
	Warnings      []string
	ExpandedStars int
}

// Changed returns true if any changes were made to the file.
func (r Report) Changed() bool {
	return len(r.Added) > 0 || r.ExpandedStars > 0
}
