package sqlite

import (
	"fmt"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

// sqlGenerator implements engine.SQLGenerator for SQLite.
type sqlGenerator struct{}

// newSQLGenerator creates a new SQLite SQL generator.
func newSQLGenerator() *sqlGenerator {
	return &sqlGenerator{}
}

// GenerateTable creates a CREATE TABLE statement for SQLite.
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

	buf.WriteString("\n)")

	// Add table options
	if table.WithoutRowID {
		buf.WriteString(" WITHOUT ROWID")
	}
	if table.Strict {
		buf.WriteString(" STRICT")
	}

	buf.WriteString(";")

	return buf.String()
}

// GenerateIndex creates a CREATE INDEX statement for SQLite.
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

// GenerateColumnDef creates a column definition clause for SQLite.
func (g *sqlGenerator) GenerateColumnDef(column *model.Column) string {
	var parts []string

	parts = append(parts, column.Name)
	parts = append(parts, g.sqliteColumnType(column.Type))

	if column.NotNull {
		parts = append(parts, "NOT NULL")
	}

	if column.Default != nil {
		parts = append(parts, "DEFAULT "+g.sqliteDefaultValue(column.Default))
	}

	return strings.Join(parts, " ")
}

// Dialect returns the target SQL dialect.
func (g *sqlGenerator) Dialect() string {
	return "sqlite"
}

// sqliteColumnType normalizes a type for SQLite.
func (g *sqlGenerator) sqliteColumnType(typ string) string {
	upperType := strings.ToUpper(typ)

	switch {
	case strings.Contains(upperType, "INTEGER"), strings.Contains(upperType, "INT"):
		return "INTEGER"
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"), strings.Contains(upperType, "CLOB"):
		return "TEXT"
	case strings.Contains(upperType, "BLOB"):
		return "BLOB"
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT"), strings.Contains(upperType, "DOUBLE"):
		return "REAL"
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"), strings.Contains(upperType, "BOOLEAN"), strings.Contains(upperType, "DATE"), strings.Contains(upperType, "DATETIME"):
		return "NUMERIC"
	default:
		return "TEXT"
	}
}

// sqliteDefaultValue converts a value to SQLite DEFAULT clause.
func (g *sqlGenerator) sqliteDefaultValue(v *model.Value) string {
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
