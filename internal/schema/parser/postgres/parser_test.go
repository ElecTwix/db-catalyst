package postgres

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
			name: "simple table with PostgreSQL types",
			ddl: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email TEXT UNIQUE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);`,
			wantTables: 1,
		},
		{
			name: "table with JSONB and arrays",
			ddl: `CREATE TABLE posts (
				id BIGSERIAL PRIMARY KEY,
				title VARCHAR(255) NOT NULL,
				content TEXT,
				metadata JSONB,
				tags TEXT[],
				created_at TIMESTAMPTZ DEFAULT NOW()
			);`,
			wantTables: 1,
		},
		{
			name: "table with UUID",
			ddl: `CREATE TABLE sessions (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id INTEGER REFERENCES users(id),
				expires_at TIMESTAMP
			);`,
			wantTables: 1,
		},
		{
			name: "multiple tables with foreign keys",
			ddl: `CREATE TABLE authors (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			);
			
			CREATE TABLE books (
				id SERIAL PRIMARY KEY,
				title VARCHAR(255) NOT NULL,
				author_id INTEGER REFERENCES authors(id)
			);`,
			wantTables: 2,
		},
		{
			name: "table with check constraint",
			ddl: `CREATE TABLE products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				price NUMERIC(10,2) CHECK (price > 0)
			);`,
			wantTables: 1,
		},
		{
			name: "create index",
			ddl: `CREATE TABLE items (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100)
			);
			
			CREATE INDEX idx_items_name ON items(name);`,
			wantTables: 1,
		},
		{
			name: "enum_definition",
			ddl: `CREATE TYPE user_status AS ENUM ('pending', 'active', 'completed');
			
			CREATE TABLE tasks (
				id SERIAL PRIMARY KEY,
				status user_status DEFAULT 'pending'
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
			wantErr:   false,          // Parser doesn't error, just produces diagnostics
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

	ddl := `CREATE TABLE test (id SERIAL PRIMARY KEY);`

	_, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err == nil {
		t.Error("Parse() expected error for cancelled context, got nil")
	}
}

func TestParser_ColumnTypes(t *testing.T) {
	parser := New()
	ctx := context.Background()

	ddl := `CREATE TABLE type_test (
		id SERIAL PRIMARY KEY,
		big_id BIGSERIAL,
		small_id SMALLSERIAL,
		name VARCHAR(255),
		description TEXT,
		count INTEGER,
		amount NUMERIC(10,2),
		price DECIMAL(8,2),
		rating REAL,
		score DOUBLE PRECISION,
		active BOOLEAN,
		created_at TIMESTAMP,
		updated_at TIMESTAMPTZ,
		birth_date DATE,
		start_time TIME,
		duration INTERVAL,
		data BYTEA,
		settings JSON,
		config JSONB,
		uuid UUID,
		ip_address INET,
		tags TEXT[],
		scores INTEGER[]
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

	// Verify all 23 columns were parsed correctly
	if len(table.Columns) != 23 {
		t.Errorf("Expected 23 columns, got %d", len(table.Columns))
	}

	// Verify specific column types
	expectedTypes := map[string]string{
		"id":          "SERIAL",
		"big_id":      "BIGSERIAL",
		"small_id":    "SMALLSERIAL",
		"name":        "VARCHAR(255)",
		"description": "TEXT",
		"count":       "INTEGER",
		"amount":      "NUMERIC(10,2)",
		"price":       "DECIMAL(8,2)",
		"rating":      "REAL",
		"score":       "DOUBLE PRECISION",
		"active":      "BOOLEAN",
		"created_at":  "TIMESTAMP",
		"updated_at":  "TIMESTAMPTZ",
		"birth_date":  "DATE",
		"start_time":  "TIME",
		"duration":    "INTERVAL",
		"data":        "BYTEA",
		"settings":    "JSON",
		"config":      "JSONB",
		"uuid":        "UUID",
		"ip_address":  "INET",
		"tags":        "TEXT[]",
		"scores":      "INTEGER[]",
	}

	for colName, wantType := range expectedTypes {
		found := false
		for _, col := range table.Columns {
			if col.Name == colName {
				found = true
				if col.Type != wantType {
					t.Errorf("Column %s: expected type %q, got %q", colName, wantType, col.Type)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected column %s not found", colName)
		}
	}
}

func TestParser_Enums(t *testing.T) {
	parser := New()
	ctx := context.Background()

	ddl := `CREATE TYPE user_status AS ENUM ('pending', 'active', 'completed');
	
	CREATE TYPE priority AS ENUM ('low', 'medium', 'high', 'urgent');
	
	CREATE TABLE tasks (
		id SERIAL PRIMARY KEY,
		status user_status DEFAULT 'pending',
		priority_level priority DEFAULT 'medium'
	);`

	catalog, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify table was parsed
	if len(catalog.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(catalog.Tables))
	}

	// Verify enums were parsed
	if len(catalog.Enums) != 2 {
		t.Errorf("Expected 2 enums, got %d", len(catalog.Enums))
	}

	// Verify user_status enum
	userStatus := catalog.Enums["user_status"]
	if userStatus == nil {
		t.Fatal("Enum 'user_status' not found")
	}
	if len(userStatus.Values) != 3 {
		t.Errorf("Expected 3 values in user_status, got %d", len(userStatus.Values))
	}

	// Verify priority enum
	priority := catalog.Enums["priority"]
	if priority == nil {
		t.Fatal("Enum 'priority' not found")
	}
	if len(priority.Values) != 4 {
		t.Errorf("Expected 4 values in priority, got %d", len(priority.Values))
	}
}

func TestParser_Domains(t *testing.T) {
	parser := New()
	ctx := context.Background()

	// Test domains separately from table parsing with custom types
	ddl := `CREATE DOMAIN email_address AS VARCHAR(255);

	CREATE DOMAIN positive_integer AS INTEGER;`

	catalog, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify domains were parsed
	if len(catalog.Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(catalog.Domains))
	}

	// Verify email_address domain
	emailDomain := catalog.Domains["email_address"]
	if emailDomain == nil {
		t.Fatal("Domain 'email_address' not found")
	}
	if emailDomain.BaseType != "VARCHAR(255)" {
		t.Errorf("Expected base type VARCHAR(255), got %s", emailDomain.BaseType)
	}

	// Verify positive_integer domain
	posIntDomain := catalog.Domains["positive_integer"]
	if posIntDomain == nil {
		t.Fatal("Domain 'positive_integer' not found")
	}
	if posIntDomain.BaseType != "INTEGER" {
		t.Errorf("Expected base type INTEGER, got %s", posIntDomain.BaseType)
	}
}

func TestParser_ConstraintValidation(t *testing.T) {
	parser := New()
	ctx := context.Background()

	ddl := `CREATE TABLE users (
		id SERIAL PRIMARY KEY
	);

	CREATE TABLE orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		product_id INTEGER,
		quantity INTEGER,
		status VARCHAR(20) DEFAULT 'pending'
	);`

	catalog, _, err := parser.Parse(ctx, "test.sql", []byte(ddl))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify both tables were parsed
	if len(catalog.Tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(catalog.Tables))
	}

	// Verify orders table exists with correct columns
	ordersTable := catalog.Tables["orders"]
	if ordersTable == nil {
		t.Fatal("Table 'orders' not found")
	}
	if len(ordersTable.Columns) != 5 {
		t.Errorf("Expected 5 columns in orders, got %d", len(ordersTable.Columns))
	}

	// Verify users table exists
	usersTable := catalog.Tables["users"]
	if usersTable == nil {
		t.Fatal("Table 'users' not found")
	}
}
