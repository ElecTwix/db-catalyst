package sqlfix

type AddedAlias struct {
	QueryName   string
	ColumnIndex int
	Alias       string
}

type SkippedAlias struct {
	QueryName string
	Expr      string
	Reason    string
}

type Report struct {
	Path          string
	Added         []AddedAlias
	Skipped       []SkippedAlias
	Warnings      []string
	ExpandedStars int
}

func (r Report) Changed() bool {
	return len(r.Added) > 0 || r.ExpandedStars > 0
}
