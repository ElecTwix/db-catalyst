package overrides

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/sqlfix/sqlcconfig"
)

// Mappings is a convenience alias for slices of custom type mappings.
type Mappings []config.CustomTypeMapping

// ConvertOverrides converts sqlc overrides into db-catalyst custom type mappings and warnings.
func ConvertOverrides(cfg sqlcconfig.Config) (Mappings, []string) {
	warnings := cfg.SchemaWarnings()
	registry := newNameRegistry()

	mappings := make([]config.CustomTypeMapping, 0, len(cfg.Overrides))
	for _, override := range cfg.Overrides {
		info, err := override.GoType.Normalize()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("override skipped: %v", err))
			continue
		}

		mapping := config.CustomTypeMapping{
			GoType:    info.TypeName,
			GoImport:  strings.TrimSpace(info.ImportPath),
			GoPackage: strings.TrimSpace(info.PackageName),
			Pointer:   info.Pointer,
		}
		if mapping.GoImport != "" && mapping.GoPackage == "" {
			mapping.GoPackage = path.Base(mapping.GoImport)
		}
		if override.DBType != "" {
			sqliteType := strings.ToUpper(strings.TrimSpace(override.DBType))
			if sqliteType == "" {
				warnings = append(warnings, "db_type override with empty db_type value skipped")
				continue
			}
			mapping.SQLiteType = sqliteType
			base := sanitizeIdentifier(info.TypeName)
			if base == "" {
				base = sanitizeIdentifier(sqliteType)
			}
			mapping.CustomType = registry.unique(base)
			mappings = append(mappings, mapping)
			continue
		}
		if override.Column.IsZero() {
			warnings = append(warnings, "override missing db_type or column; skipped")
			continue
		}
		columnRef := sqlcconfig.ColumnRef{
			Schema: override.Column.Schema,
			Table:  override.Column.Table,
			Column: override.Column.Name,
		}
		sqliteType, ok := cfg.ColumnType(columnRef)
		if !ok || sqliteType == "" {
			warnings = append(warnings, fmt.Sprintf("column override %s.%s: SQLite type not found; skipped", override.Column.Table, override.Column.Name))
			continue
		}
		mapping.SQLiteType = sqliteType
		base := buildColumnCustomTypeBase(override.Column, info.TypeName)
		mapping.CustomType = registry.unique(sanitizeIdentifier(base))
		mappings = append(mappings, mapping)
	}

	sort.Strings(warnings)
	return mappings, warnings
}

// MergeMappings merges additions into existing mappings, preserving existing entries on conflict.
func MergeMappings(existing []config.CustomTypeMapping, additions Mappings) ([]config.CustomTypeMapping, []string) {
	if len(additions) == 0 {
		return append([]config.CustomTypeMapping(nil), existing...), nil
	}

	result := append([]config.CustomTypeMapping(nil), existing...)
	index := make(map[string]config.CustomTypeMapping, len(existing))
	for _, mapping := range existing {
		index[strings.ToLower(mapping.CustomType)] = mapping
	}

	warnings := make([]string, 0)
	for _, mapping := range additions {
		key := strings.ToLower(mapping.CustomType)
		if existing, ok := index[key]; ok {
			if !equivalentMapping(existing, mapping) {
				warnings = append(warnings, fmt.Sprintf("custom type %s already defined; keeping existing mapping", mapping.CustomType))
			}
			continue
		}
		result = append(result, mapping)
		index[key] = mapping
	}

	return result, warnings
}

func equivalentMapping(a, b config.CustomTypeMapping) bool {
	return a.CustomType == b.CustomType &&
		a.SQLiteType == b.SQLiteType &&
		a.GoType == b.GoType &&
		a.GoImport == b.GoImport &&
		a.GoPackage == b.GoPackage &&
		a.Pointer == b.Pointer
}

func buildColumnCustomTypeBase(target sqlcconfig.ColumnTarget, typeName string) string {
	parts := make([]string, 0, 4)
	if target.Schema != "" {
		parts = append(parts, target.Schema)
	}
	if target.Table != "" {
		parts = append(parts, target.Table)
	}
	if target.Name != "" {
		parts = append(parts, target.Name)
	}
	if typeName != "" {
		parts = append(parts, typeName)
	}
	return strings.Join(parts, "_")
}

type nameRegistry struct {
	counts map[string]int
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{counts: make(map[string]int)}
}

func (n *nameRegistry) unique(base string) string {
	if base == "" {
		base = "custom_type"
	}
	if _, ok := n.counts[base]; !ok {
		n.counts[base] = 1
		return base
	}
	idx := n.counts[base]
	for {
		candidate := fmt.Sprintf("%s_%d", base, idx)
		if _, exists := n.counts[candidate]; !exists {
			n.counts[base] = idx + 1
			n.counts[candidate] = 1
			return candidate
		}
		idx++
	}
}

func sanitizeIdentifier(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(text))
	lastUnderscore := false
	for _, r := range text {
		switch {
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
			lastUnderscore = false
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			if builder.Len() == 0 {
				builder.WriteRune('_')
			}
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && builder.Len() > 0 {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	out := builder.String()
	out = strings.Trim(out, "_")
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	return out
}
