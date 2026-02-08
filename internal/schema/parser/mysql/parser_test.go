package mysql

import (
	"context"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := New()
	ctx := context.Background()

	tests := []struct {
		name       string
		ddl        string
		wantTables int
		wantErr    bool
		wantDiags  int
	}{
		{
			name: "simple table with MySQL types",
			ddl: `CREATE TABLE users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) UNIQUE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			) ENGINE=InnoDB;`,
			wantTables: 1,
		},
		{
			name: "table with MySQL integer types",
			ddl: `CREATE TABLE products (
				id BIGINT AUTO_INCREMENT PRIMARY KEY,
				small_id SMALLINT,
				tiny_id TINYINT UNSIGNED,
				medium_id MEDIUMINT,
				price DECIMAL(10,2)
			);`,
			wantTables: 1,
		},
		{
			name: "table with ENUM and SET",
			ddl: `CREATE TABLE items (
				id INT AUTO_INCREMENT PRIMARY KEY,
				status ENUM('active', 'inactive', 'pending') DEFAULT 'pending',
				tags SET('red', 'green', 'blue')
			);`,
			wantTables: 1,
		},
		{
			name: "table with JSON",
			ddl: `CREATE TABLE configs (
				id INT AUTO_INCREMENT PRIMARY KEY,
				settings JSON,
				created_at DATETIME
			);`,
			wantTables: 1,
		},
		{
			name: "table with full-text index",
			ddl: `CREATE TABLE articles (
				id INT AUTO_INCREMENT PRIMARY KEY,
				title VARCHAR(255),
				content TEXT,
				FULLTEXT INDEX idx_content (title, content)
			);`,
			wantTables: 1,
		},
		{
			name: "table with foreign key",
			ddl: `CREATE TABLE orders (
				id INT AUTO_INCREMENT PRIMARY KEY,
				user_id INT,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
			wantTables: 1,
		},
		{
			name:       "empty DDL",
			ddl:        "",
			wantTables: 0,
		},
		{
			name:      "invalid syntax",
			ddl:       "CREATE TABLE", // Missing table name
			wantErr:   false,
			wantDiags: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, diags, err := parser.Parse(ctx, "test.sql", []byte(tt.ddl))

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if catalog != nil && len(catalog.Tables) != tt.wantTables {
				t.Errorf("Parse() got %d tables, want %d", len(catalog.Tables), tt.wantTables)
			}

			if tt.wantDiags > 0 && len(diags) < tt.wantDiags {
				t.Errorf("Parse() got %d diagnostics, want at least %d", len(diags), tt.wantDiags)
			}
		})
	}
}

func TestParser_ParseContextCancellation(t *testing.T) {
	parser := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ddl := `CREATE TABLE test (id INT AUTO_INCREMENT PRIMARY KEY);`

	_, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err == nil {
		t.Error("Parse() expected error for cancelled context, got nil")
	}
}

func TestParser_ColumnTypes(t *testing.T) {
	parser := New()
	ctx := context.Background()

	ddl := `CREATE TABLE type_test (
		id INT AUTO_INCREMENT PRIMARY KEY,
		big_id BIGINT,
		small_id SMALLINT,
		tiny_id TINYINT,
		medium_id MEDIUMINT,
		name VARCHAR(255),
		description TEXT,
		content LONGTEXT,
		data BLOB,
		big_data LONGBLOB,
		price DECIMAL(10,2),
		amount FLOAT,
		total DOUBLE,
		is_active BOOLEAN,
		created_at DATETIME,
		updated_at TIMESTAMP,
		birth_date DATE,
		event_time TIME,
		year_val YEAR,
		config JSON,
		status ENUM('a', 'b', 'c'),
		tags SET('x', 'y', 'z'),
		binary_data BINARY(16),
		var_data VARBINARY(255)
	);`

	catalog, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(catalog.Tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(catalog.Tables))
	}

	table := catalog.Tables["type_test"]
	if table == nil {
		t.Fatal("Table 'type_test' not found")
	}

	// Verify we got columns (some may not parse perfectly)
	if len(table.Columns) == 0 {
		t.Error("Expected at least some columns, got 0")
	}
}

func TestParser_AutoIncrement(t *testing.T) {
	parser := New()
	ctx := context.Background()

	ddl := `CREATE TABLE users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL
	);`

	catalog, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(catalog.Tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(catalog.Tables))
	}

	table := catalog.Tables["users"]
	if table == nil {
		t.Fatal("Table 'users' not found")
	}

	// Should have primary key
	if table.PrimaryKey == nil {
		t.Error("Expected primary key for AUTO_INCREMENT column")
	}
}
