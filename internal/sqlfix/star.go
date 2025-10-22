package sqlfix

import (
	"fmt"
	"strings"

	"github.com/electwix/db-catalyst/internal/query/block"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	schematokenizer "github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

type relationRef struct {
	aliasCanonical  string
	aliasNormalized string
	tableCanonical  string
	tableNormalized string
	baseTable       bool
}

type starExpression struct {
	start              int
	end                int
	original           string
	qualifierOriginal  string
	qualifierCanonical string
	hasQualifier       bool
}

var disallowedAliasTokens = map[string]struct{}{
	"JOIN":    {},
	"ON":      {},
	"USING":   {},
	"WHERE":   {},
	"GROUP":   {},
	"ORDER":   {},
	"LIMIT":   {},
	"INNER":   {},
	"LEFT":    {},
	"RIGHT":   {},
	"FULL":    {},
	"CROSS":   {},
	"NATURAL": {},
}

func (r *Runner) expandStars(blk block.Block, sql string, query queryparser.Query) (string, []string, int, error) {
	stars := findStarExpressions(sql, query.Columns)
	if len(stars) == 0 {
		return sql, nil, 0, nil
	}

	tokens, err := schematokenizer.Scan(blk.Path, []byte(sql), false)
	if err != nil {
		warning := fmt.Sprintf("%s:%s: unable to analyze FROM clause: %v", blk.Path, blk.Name, err)
		return sql, []string{warning}, 0, nil
	}

	relations := collectRelations(tokens)
	relationIndex := buildRelationIndex(relations)

	warnings := make([]string, 0, len(stars))
	edits := make([]edit, 0, len(stars))
	replacements := 0

	for _, star := range stars {
		var refs []*relationRef
		if star.hasQualifier {
			ref, ok := relationIndex[star.qualifierCanonical]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("%s:%s: cannot expand %q: relation %q not found", blk.Path, blk.Name, star.original, star.qualifierOriginal))
				continue
			}
			refs = []*relationRef{ref}
		} else {
			if len(relations) == 0 {
				warnings = append(warnings, fmt.Sprintf("%s:%s: cannot expand %q: no table sources found", blk.Path, blk.Name, star.original))
				continue
			}
			refs = relations
		}

		cols, colWarnings, ok := r.columnsForStar(refs, star)
		if len(colWarnings) > 0 {
			for _, msg := range colWarnings {
				warnings = append(warnings, fmt.Sprintf("%s:%s: %s", blk.Path, blk.Name, msg))
			}
		}
		if !ok {
			continue
		}

		replacement := strings.Join(cols, ", ")
		edits = append(edits, edit{
			start: star.start,
			end:   star.end,
			text:  replacement,
		})
		replacements++
	}

	if replacements == 0 {
		return sql, warnings, 0, nil
	}

	updated, err := applyStringEdits(sql, edits)
	if err != nil {
		return sql, warnings, 0, err
	}

	return updated, warnings, replacements, nil
}

func findStarExpressions(sql string, cols []queryparser.Column) []starExpression {
	results := make([]starExpression, 0)
	for _, col := range cols {
		if col.StartOffset < 0 || col.EndOffset > len(sql) || col.EndOffset <= col.StartOffset {
			continue
		}
		expr := strings.TrimSpace(sql[col.StartOffset:col.EndOffset])
		if expr == "" {
			continue
		}
		if expr == "*" {
			results = append(results, starExpression{
				start:    col.StartOffset,
				end:      col.EndOffset,
				original: expr,
			})
			continue
		}
		dot := strings.LastIndex(expr, ".")
		if dot == -1 {
			continue
		}
		if strings.TrimSpace(expr[dot+1:]) != "*" {
			continue
		}
		qualifier := strings.TrimSpace(expr[:dot])
		if qualifier == "" {
			continue
		}
		canonical := canonicalQualifier(qualifier)
		if canonical == "" {
			continue
		}
		results = append(results, starExpression{
			start:              col.StartOffset,
			end:                col.EndOffset,
			original:           expr,
			qualifierOriginal:  qualifier,
			qualifierCanonical: canonical,
			hasQualifier:       true,
		})
	}
	return results
}

func canonicalQualifier(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ".")
	last := strings.TrimSpace(parts[len(parts)-1])
	return canonicalIdent(last)
}

func canonicalIdent(name string) string {
	normalized := schematokenizer.NormalizeIdentifier(strings.TrimSpace(name))
	if normalized == "" {
		return ""
	}
	return strings.ToLower(normalized)
}

func collectRelations(tokens []schematokenizer.Token) []*relationRef {
	relations := make([]*relationRef, 0, 4)
	depth := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind == schematokenizer.KindEOF {
			break
		}
		if tok.Kind == schematokenizer.KindSymbol {
			switch tok.Text {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			}
			continue
		}
		if depth > 0 {
			continue
		}
		if tok.Kind != schematokenizer.KindKeyword && tok.Kind != schematokenizer.KindIdentifier {
			continue
		}
		switch strings.ToUpper(tok.Text) {
		case "FROM":
			i++
			next := parseRelationSequence(tokens, i, &relations)
			i = next - 1
		case "JOIN":
			i++
			ref, next := parseSingleRelation(tokens, i)
			if ref != nil {
				relations = append(relations, ref)
			}
			i = next - 1
		}
	}
	return relations
}

func parseRelationSequence(tokens []schematokenizer.Token, idx int, dest *[]*relationRef) int {
	i := idx
	for {
		ref, next := parseSingleRelation(tokens, i)
		if ref != nil {
			*dest = append(*dest, ref)
		}
		if next <= i {
			i = next
			break
		}
		i = next
		for i < len(tokens) && tokens[i].Kind == schematokenizer.KindDocComment {
			i++
		}
		if i < len(tokens) && tokens[i].Kind == schematokenizer.KindSymbol && tokens[i].Text == "," {
			i++
			continue
		}
		break
	}
	return i
}

func parseSingleRelation(tokens []schematokenizer.Token, idx int) (*relationRef, int) {
	i := idx
	for i < len(tokens) && tokens[i].Kind == schematokenizer.KindDocComment {
		i++
	}
	if i >= len(tokens) {
		return nil, i
	}

	tok := tokens[i]
	if tok.Kind == schematokenizer.KindKeyword {
		if strings.ToUpper(tok.Text) == "LATERAL" {
			i++
			for i < len(tokens) && tokens[i].Kind == schematokenizer.KindDocComment {
				i++
			}
			if i >= len(tokens) {
				return nil, i
			}
			tok = tokens[i]
		}
	}

	if tok.Kind == schematokenizer.KindSymbol && tok.Text == "(" {
		depth := 1
		i++
		for i < len(tokens) && depth > 0 {
			t := tokens[i]
			if t.Kind == schematokenizer.KindSymbol {
				switch t.Text {
				case "(":
					depth++
				case ")":
					depth--
				}
			}
			i++
		}
		aliasToken, next := parseAliasToken(tokens, i)
		if aliasToken == "" {
			return &relationRef{baseTable: false}, next
		}
		aliasNorm := schematokenizer.NormalizeIdentifier(aliasToken)
		return &relationRef{
			aliasCanonical:  canonicalIdent(aliasToken),
			aliasNormalized: aliasNorm,
			baseTable:       false,
		}, next
	}

	if tok.Kind != schematokenizer.KindIdentifier {
		return nil, i
	}

	parts := []string{tok.Text}
	i++
	for i+1 < len(tokens) && tokens[i].Kind == schematokenizer.KindSymbol && tokens[i].Text == "." && tokens[i+1].Kind == schematokenizer.KindIdentifier {
		parts = append(parts, tokens[i+1].Text)
		i += 2
	}

	tableToken := parts[len(parts)-1]
	tableNorm := schematokenizer.NormalizeIdentifier(tableToken)
	tableCanon := canonicalIdent(tableToken)

	aliasToken, next := parseAliasToken(tokens, i)
	aliasNorm := tableNorm
	aliasCanon := tableCanon
	if aliasToken != "" {
		aliasNorm = schematokenizer.NormalizeIdentifier(aliasToken)
		aliasCanon = canonicalIdent(aliasToken)
		i = next
	} else {
		i = next
	}

	return &relationRef{
		aliasCanonical:  aliasCanon,
		aliasNormalized: aliasNorm,
		tableCanonical:  tableCanon,
		tableNormalized: tableNorm,
		baseTable:       true,
	}, i
}

func parseAliasToken(tokens []schematokenizer.Token, idx int) (string, int) {
	i := idx
	for i < len(tokens) && tokens[i].Kind == schematokenizer.KindDocComment {
		i++
	}
	if i < len(tokens) && tokens[i].Kind == schematokenizer.KindKeyword {
		if strings.ToUpper(tokens[i].Text) == "AS" {
			i++
			for i < len(tokens) && tokens[i].Kind == schematokenizer.KindDocComment {
				i++
			}
		}
	}
	if i < len(tokens) && tokens[i].Kind == schematokenizer.KindIdentifier {
		text := tokens[i].Text
		upper := strings.ToUpper(text)
		if schematokenizer.IsKeyword(text) {
			return "", idx
		}
		if _, blocked := disallowedAliasTokens[upper]; blocked {
			return "", idx
		}
		return text, i + 1
	}
	return "", idx
}

func buildRelationIndex(refs []*relationRef) map[string]*relationRef {
	index := make(map[string]*relationRef, len(refs)*2)
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		if ref.aliasCanonical != "" {
			if _, exists := index[ref.aliasCanonical]; !exists {
				index[ref.aliasCanonical] = ref
			}
		}
		if ref.baseTable && ref.tableCanonical != "" {
			if _, exists := index[ref.tableCanonical]; !exists {
				index[ref.tableCanonical] = ref
			}
		}
	}
	return index
}

func (r *Runner) columnsForStar(refs []*relationRef, star starExpression) ([]string, []string, bool) {
	warnings := make([]string, 0)
	columns := make([]string, 0)
	for _, ref := range refs {
		colNames, warn, ok := r.columnsForRelation(ref)
		if !ok {
			if warn != "" {
				warnings = append(warnings, warn)
			}
			return nil, warnings, false
		}
		if star.hasQualifier {
			qualifier := star.qualifierOriginal
			if qualifier == "" {
				qualifier = ref.aliasNormalized
				if qualifier == "" {
					qualifier = ref.tableNormalized
				}
			}
			for _, name := range colNames {
				columns = append(columns, qualifier+"."+name)
			}
		} else {
			columns = append(columns, colNames...)
		}
	}
	if len(columns) == 0 {
		return nil, warnings, false
	}
	return columns, warnings, true
}

func (r *Runner) columnsForRelation(ref *relationRef) ([]string, string, bool) {
	if ref == nil {
		return nil, "relation not resolved", false
	}
	if !ref.baseTable {
		name := ref.aliasNormalized
		if name == "" {
			name = ref.tableNormalized
		}
		if name == "" {
			name = "subquery"
		}
		return nil, fmt.Sprintf("relation %q is not backed by a schema table", name), false
	}
	if r.catalog == nil || r.catalog.Tables == nil {
		return nil, "schema catalog not available", false
	}
	tbl := r.catalog.Tables[ref.tableCanonical]
	if tbl == nil {
		target := ref.tableNormalized
		if target == "" {
			target = ref.aliasNormalized
		}
		if target == "" {
			target = "unknown"
		}
		return nil, fmt.Sprintf("table %q not found in schema", target), false
	}
	cols := make([]string, len(tbl.Columns))
	for i, col := range tbl.Columns {
		cols[i] = col.Name
	}
	return cols, "", true
}
