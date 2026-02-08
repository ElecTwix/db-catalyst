package postgres

import (
	"fmt"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

// sqlGenerator implements engine.SQLGenerator for PostgreSQL.
type sqlGenerator struct{}

// GenerateTable creates a CREATE TABLE statement for PostgreSQL.
func (g *sqlGenerator) GenerateTable(table *model.Table) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name))

	for i, col := range table.Columns {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString("    ")
		buf.WriteString(g.GenerateColumnDef(col))
	}

	// Add primary key constraint if defined at table level
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) > 0 {
		buf.WriteString(",\n    PRIMARY KEY (")
		buf.WriteString(strings.Join(table.PrimaryKey.Columns, ", "))
		buf.WriteString(")")
	}

	// Add unique constraints
	for _, uk := range table.UniqueKeys {
		if uk.Name != "" {
			buf.WriteString(fmt.Sprintf(",\n    CONSTRAINT %s UNIQUE (", uk.Name))
		} else {
			buf.WriteString(",\n    UNIQUE (")
		}
		buf.WriteString(strings.Join(uk.Columns, ", "))
		buf.WriteString(")")
	}

	// Add foreign key constraints
	for _, fk := range table.ForeignKeys {
		if fk.Name != "" {
			buf.WriteString(fmt.Sprintf(",\n    CONSTRAINT %s FOREIGN KEY (", fk.Name))
		} else {
			buf.WriteString(",\n    FOREIGN KEY (")
		}
		buf.WriteString(strings.Join(fk.Columns, ", "))
		buf.WriteString(") REFERENCES ")
		buf.WriteString(fk.Ref.Table)
		if len(fk.Ref.Columns) > 0 {
			buf.WriteString("(")
			buf.WriteString(strings.Join(fk.Ref.Columns, ", "))
			buf.WriteString(")")
		}
		buf.WriteString(")")
	}

	buf.WriteString("\n);")

	return buf.String()
}

// GenerateIndex creates a CREATE INDEX statement for PostgreSQL.
func (g *sqlGenerator) GenerateIndex(index *model.Index, tableName string) string {
	var buf strings.Builder

	if index.Unique {
		buf.WriteString("CREATE UNIQUE INDEX ")
	} else {
		buf.WriteString("CREATE INDEX ")
	}

	buf.WriteString(index.Name)
	buf.WriteString(" ON ")
	buf.WriteString(tableName)
	buf.WriteString(" (")
	buf.WriteString(strings.Join(index.Columns, ", "))
	buf.WriteString(");")

	return buf.String()
}

// GenerateColumnDef creates a column definition clause for PostgreSQL.
func (g *sqlGenerator) GenerateColumnDef(column *model.Column) string {
	var parts []string

	parts = append(parts, column.Name)
	parts = append(parts, g.postgresColumnType(column.Type))

	if column.NotNull {
		parts = append(parts, "NOT NULL")
	}

	if column.Default != nil {
		parts = append(parts, "DEFAULT "+g.postgresDefaultValue(column.Default))
	}

	return strings.Join(parts, " ")
}

// Dialect returns the target SQL dialect.
func (g *sqlGenerator) Dialect() string {
	return "postgresql"
}

// postgresColumnType converts SQLite type to PostgreSQL type.
func (g *sqlGenerator) postgresColumnType(typ string) string {
	upperType := strings.ToUpper(typ)

	switch {
	case strings.Contains(upperType, "INTEGER"), strings.Contains(upperType, "INT"):
		if strings.Contains(upperType, "BIG") {
			return "BIGINT"
		}
		if strings.Contains(upperType, "SMALL") {
			return "SMALLINT"
		}
		return "INTEGER"
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"):
		if strings.Contains(upperType, "VARCHAR") {
			// Extract size if present
			return typ
		}
		return "TEXT"
	case strings.Contains(upperType, "BLOB"), strings.Contains(upperType, "BYTEA"):
		return "BYTEA"
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT"), strings.Contains(upperType, "DOUBLE"):
		return "DOUBLE PRECISION"
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"):
		return "NUMERIC"
	case strings.Contains(upperType, "BOOLEAN"), strings.Contains(upperType, "BOOL"):
		return "BOOLEAN"
	case strings.Contains(upperType, "TIMESTAMP"):
		if strings.Contains(upperType, "TZ") || strings.Contains(upperType, "ZONE") {
			return "TIMESTAMPTZ"
		}
		return "TIMESTAMP"
	case strings.Contains(upperType, "DATE"):
		return "DATE"
	case strings.Contains(upperType, "UUID"):
		return "UUID"
	case strings.Contains(upperType, "JSON"):
		if strings.Contains(upperType, "JSONB") {
			return "JSONB"
		}
		return "JSON"
	default:
		return "TEXT"
	}
}

// postgresDefaultValue converts a value to PostgreSQL DEFAULT clause.
func (g *sqlGenerator) postgresDefaultValue(v *model.Value) string {
	switch v.Kind {
	case model.ValueKindNumber:
		return v.Text
	case model.ValueKindString:
		return v.Text
	case model.ValueKindKeyword:
		return v.Text
	default:
		return v.Text
	}
}
