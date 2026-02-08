package mysql

import (
	"fmt"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

// sqlGenerator implements engine.SQLGenerator for MySQL.
type sqlGenerator struct{}

// GenerateTable creates a CREATE TABLE statement for MySQL.
func (g *sqlGenerator) GenerateTable(table *model.Table) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", table.Name))

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
		quotedCols := make([]string, len(table.PrimaryKey.Columns))
		for i, col := range table.PrimaryKey.Columns {
			quotedCols[i] = fmt.Sprintf("`%s`", col)
		}
		buf.WriteString(strings.Join(quotedCols, ", "))
		buf.WriteString(")")
	}

	// Add unique constraints
	for _, uk := range table.UniqueKeys {
		if uk.Name != "" {
			buf.WriteString(fmt.Sprintf(",\n    CONSTRAINT `%s` UNIQUE (", uk.Name))
		} else {
			buf.WriteString(",\n    UNIQUE (")
		}
		quotedCols := make([]string, len(uk.Columns))
		for i, col := range uk.Columns {
			quotedCols[i] = fmt.Sprintf("`%s`", col)
		}
		buf.WriteString(strings.Join(quotedCols, ", "))
		buf.WriteString(")")
	}

	// Add foreign key constraints
	for _, fk := range table.ForeignKeys {
		if fk.Name != "" {
			buf.WriteString(fmt.Sprintf(",\n    CONSTRAINT `%s` FOREIGN KEY (", fk.Name))
		} else {
			buf.WriteString(",\n    FOREIGN KEY (")
		}
		quotedCols := make([]string, len(fk.Columns))
		for i, col := range fk.Columns {
			quotedCols[i] = fmt.Sprintf("`%s`", col)
		}
		buf.WriteString(strings.Join(quotedCols, ", "))
		buf.WriteString(") REFERENCES ")
		buf.WriteString(fmt.Sprintf("`%s`", fk.Ref.Table))
		if len(fk.Ref.Columns) > 0 {
			quotedRefCols := make([]string, len(fk.Ref.Columns))
			for i, col := range fk.Ref.Columns {
				quotedRefCols[i] = fmt.Sprintf("`%s`", col)
			}
			buf.WriteString("(")
			buf.WriteString(strings.Join(quotedRefCols, ", "))
			buf.WriteString(")")
		}
		buf.WriteString(")")
	}

	buf.WriteString("\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;")

	return buf.String()
}

// GenerateIndex creates a CREATE INDEX statement for MySQL.
func (g *sqlGenerator) GenerateIndex(index *model.Index, tableName string) string {
	var buf strings.Builder

	switch {
	case index.Unique:
		buf.WriteString("CREATE UNIQUE INDEX ")
	case isFullTextIndex(index):
		buf.WriteString("CREATE FULLTEXT INDEX ")
	default:
		buf.WriteString("CREATE INDEX ")
	}

	buf.WriteString(fmt.Sprintf("`%s` ON `%s` (", index.Name, tableName))
	quotedCols := make([]string, len(index.Columns))
	for i, col := range index.Columns {
		quotedCols[i] = fmt.Sprintf("`%s`", col)
	}
	buf.WriteString(strings.Join(quotedCols, ", "))
	buf.WriteString(");")

	return buf.String()
}

// GenerateColumnDef creates a column definition clause for MySQL.
func (g *sqlGenerator) GenerateColumnDef(column *model.Column) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("`%s`", column.Name))
	parts = append(parts, g.mysqlColumnType(column.Type))

	if column.NotNull {
		parts = append(parts, "NOT NULL")
	} else {
		parts = append(parts, "NULL")
	}

	if column.Default != nil {
		parts = append(parts, "DEFAULT "+g.mysqlDefaultValue(column.Default))
	}

	return strings.Join(parts, " ")
}

// Dialect returns the target SQL dialect.
func (g *sqlGenerator) Dialect() string {
	return "mysql"
}

// isFullTextIndex checks if this should be a FULLTEXT index.
func isFullTextIndex(index *model.Index) bool {
	// Check for fulltext indicator in index name or comment
	return strings.Contains(strings.ToLower(index.Name), "fulltext") ||
		strings.Contains(strings.ToLower(index.Name), "search")
}

// mysqlColumnType normalizes a type for MySQL.
func (g *sqlGenerator) mysqlColumnType(typ string) string {
	upperType := strings.ToUpper(typ)

	switch {
	// Integer types with display width
	case strings.Contains(upperType, "TINYINT"):
		return "TINYINT"
	case strings.Contains(upperType, "SMALLINT"):
		return "SMALLINT"
	case strings.Contains(upperType, "MEDIUMINT"):
		return "MEDIUMINT"
	case strings.Contains(upperType, "INT") || strings.Contains(upperType, "INTEGER"):
		return "INT"
	case strings.Contains(upperType, "BIGINT"):
		return "BIGINT"

	// Float types
	case strings.Contains(upperType, "FLOAT"):
		return "FLOAT"
	case strings.Contains(upperType, "DOUBLE"):
		return "DOUBLE"
	case strings.Contains(upperType, "DECIMAL"), strings.Contains(upperType, "DEC"), strings.Contains(upperType, "NUMERIC"):
		// Keep precision/scale if present
		if strings.Contains(upperType, "(") {
			return typ
		}
		return "DECIMAL(10,2)"

	// String types
	case strings.Contains(upperType, "VARCHAR"):
		if strings.Contains(upperType, "(") {
			return typ
		}
		return "VARCHAR(255)"
	case strings.Contains(upperType, "CHAR"):
		if strings.Contains(upperType, "(") {
			return typ
		}
		return "CHAR(1)"
	case strings.Contains(upperType, "TINYTEXT"):
		return "TINYTEXT"
	case strings.Contains(upperType, "TEXT"):
		return "TEXT"
	case strings.Contains(upperType, "MEDIUMTEXT"):
		return "MEDIUMTEXT"
	case strings.Contains(upperType, "LONGTEXT"):
		return "LONGTEXT"

	// Blob types
	case strings.Contains(upperType, "TINYBLOB"):
		return "TINYBLOB"
	case strings.Contains(upperType, "BLOB"):
		return "BLOB"
	case strings.Contains(upperType, "MEDIUMBLOB"):
		return "MEDIUMBLOB"
	case strings.Contains(upperType, "LONGBLOB"):
		return "LONGBLOB"
	case strings.Contains(upperType, "BINARY"):
		if strings.Contains(upperType, "(") {
			return typ
		}
		return "BINARY(1)"
	case strings.Contains(upperType, "VARBINARY"):
		if strings.Contains(upperType, "(") {
			return typ
		}
		return "VARBINARY(255)"

	// Boolean
	case strings.Contains(upperType, "BOOLEAN") || strings.Contains(upperType, "BOOL"):
		return "BOOLEAN"

	// Temporal types
	case strings.Contains(upperType, "DATE"):
		return "DATE"
	case strings.Contains(upperType, "DATETIME"):
		return "DATETIME"
	case strings.Contains(upperType, "TIMESTAMP"):
		return "TIMESTAMP"
	case strings.Contains(upperType, "TIME"):
		return "TIME"
	case strings.Contains(upperType, "YEAR"):
		return "YEAR"

	// JSON
	case strings.Contains(upperType, "JSON"):
		return "JSON"

	default:
		return typ
	}
}

// mysqlDefaultValue converts a value to MySQL DEFAULT clause.
func (g *sqlGenerator) mysqlDefaultValue(v *model.Value) string {
	switch v.Kind {
	case model.ValueKindNumber:
		return v.Text
	case model.ValueKindString:
		return v.Text
	case model.ValueKindKeyword:
		upper := strings.ToUpper(v.Text)
		if upper == "NULL" || upper == "TRUE" || upper == "FALSE" ||
			upper == "CURRENT_TIMESTAMP" || strings.Contains(upper, "CURRENT_TIMESTAMP") {
			return v.Text
		}
		return fmt.Sprintf("'%s'", v.Text)
	default:
		return v.Text
	}
}
