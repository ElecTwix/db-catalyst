package sqlfix

import (
	"fmt"
	"sort"
	"strings"
)

func applyStringEdits(src string, edits []edit) (string, error) {
	if len(edits) == 0 {
		return src, nil
	}

	ordered := make([]edit, len(edits))
	copy(ordered, edits)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].start == ordered[j].start {
			return ordered[i].end < ordered[j].end
		}
		return ordered[i].start < ordered[j].start
	})

	var b strings.Builder
	cursor := 0
	for _, e := range ordered {
		if e.start < 0 || e.start > len(src) || e.end < e.start || e.end > len(src) {
			return "", fmt.Errorf("invalid edit range [%d,%d)", e.start, e.end)
		}
		if e.start < cursor {
			return "", fmt.Errorf("overlapping edit starting at %d", e.start)
		}
		b.WriteString(src[cursor:e.start])
		b.WriteString(e.text)
		cursor = e.end
	}
	b.WriteString(src[cursor:])
	return b.String(), nil
}
