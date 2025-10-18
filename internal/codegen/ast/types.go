package ast

import "strings"

type typeInfo struct {
	GoType      string
	UsesSQLNull bool
}

func resolveType(goType string, nullable bool) typeInfo {
	base := strings.TrimSpace(goType)
	if base == "" {
		base = "interface{}"
	}
	if nullable {
		switch base {
		case "int64":
			return typeInfo{GoType: "sql.NullInt64", UsesSQLNull: true}
		case "float64":
			return typeInfo{GoType: "sql.NullFloat64", UsesSQLNull: true}
		case "string":
			return typeInfo{GoType: "sql.NullString", UsesSQLNull: true}
		case "bool":
			return typeInfo{GoType: "sql.NullBool", UsesSQLNull: true}
		default:
			// fall back to base type for nullable blobs or custom Go types
		}
	}
	return typeInfo{GoType: base, UsesSQLNull: strings.HasPrefix(base, "sql.Null")}
}
