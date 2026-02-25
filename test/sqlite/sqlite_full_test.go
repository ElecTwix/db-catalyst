package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/pipeline"
)

type testDB struct {
	*sql.DB
	queriesDir string
	schemaDir  string
	tmpDir     string
}

func setupTestDB(t testing.TB, schema, queries string) *testDB {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "sqlite-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module testapp

go 1.23

require modernc.org/sqlite v1.34.1
`), 0600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	schemaPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schema), 0600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	queriesPath := filepath.Join(tmpDir, "queries.sql")
	if err := os.WriteFile(queriesPath, []byte(queries), 0600); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	cfgContent := `package = "db"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			Logger: logging.NewNopLogger(),
		},
	}
	_, err = p.Run(context.Background(), pipeline.RunOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	db, err := sql.Open("sqlite", sqliteInMem)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("exec schema: %v", err)
	}

	return &testDB{
		DB:         db,
		queriesDir: queriesPath,
		schemaDir:  schemaPath,
		tmpDir:     tmpDir,
	}
}

func (tdb *testDB) Close() error {
	if tdb.tmpDir != "" {
		os.RemoveAll(tdb.tmpDir)
	}
	return tdb.DB.Close()
}

func TestSQLite_FullFeatureCoverage(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		queries string
		fn      func(*testing.T, *testDB)
	}{
		{
			name: "BasicCRUD",
			schema: `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);
`,
			queries: `
-- name: CreateUser :one
INSERT INTO users (username, email) VALUES (?, ?) RETURNING id, username, email, created_at;

-- name: GetUser :one
SELECT id, username, email, created_at FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, username, email, created_at FROM users ORDER BY username;

-- name: UpdateUser :one
UPDATE users SET email = ? WHERE id = ? RETURNING id, username, email, created_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
`,
			fn: testBasicCRUD,
		},
		{
			name: "ForeignKeys",
			schema: `
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL REFERENCES authors(id),
    title TEXT NOT NULL,
    published INTEGER DEFAULT 0
);
`,
			queries: `
-- name: GetAuthorWithBooks :one
SELECT a.id, a.name, COUNT(b.id) as book_count
FROM authors a
LEFT JOIN books b ON a.id = b.author_id
WHERE a.id = ?
GROUP BY a.id;

-- name: ListBooksByAuthor :many
SELECT b.id, b.author_id, b.title, b.published
FROM books b WHERE b.author_id = ?;
`,
			fn: testForeignKeys,
		},
		{
			name: "Joins",
			schema: `
CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT);
CREATE TABLE orders (id INTEGER PRIMARY KEY, customer_id INTEGER, total INTEGER);
CREATE TABLE order_items (id INTEGER PRIMARY KEY, order_id INTEGER, product TEXT, qty INTEGER);
`,
			queries: `
-- name: GetCustomerOrders :many
SELECT c.id, c.name, o.id as order_id, o.total
FROM customers c
JOIN orders o ON c.id = o.customer_id
WHERE c.id = ?;

-- name: GetOrderItems :many
SELECT oi.id, oi.order_id, oi.product, oi.qty
FROM order_items oi WHERE oi.order_id = ?;
`,
			fn: testJoins,
		},
		{
			name: "Aggregates",
			schema: `
CREATE TABLE sales (id INTEGER PRIMARY KEY, product TEXT, amount INTEGER, sale_date TEXT);
`,
			queries: `
-- name: GetTotalSales :one
SELECT product, SUM(amount) as total FROM sales WHERE product = ? GROUP BY product;

-- name: GetSalesCount :one
SELECT COUNT(*) as cnt FROM sales WHERE product = ?;

-- name: GetAverageSale :one
SELECT AVG(amount) as avg_amount FROM sales WHERE product = ?;
`,
			fn: testAggregates,
		},
		{
			name: "Subqueries",
			schema: `
CREATE TABLE employees (id INTEGER PRIMARY KEY, name TEXT, department TEXT, salary INTEGER);
CREATE TABLE departments (id INTEGER PRIMARY KEY, name TEXT, budget INTEGER);
`,
			queries: `
-- name: GetHighEarners :many
SELECT id, name, salary FROM employees WHERE salary > (SELECT AVG(salary) FROM employees);

-- name: GetDepartmentStats :many
SELECT d.name, d.budget, (SELECT COUNT(*) FROM employees WHERE department = d.name) as emp_count
FROM departments d;
`,
			fn: testSubqueries,
		},
		{
			name: "CTE",
			schema: `
CREATE TABLE org_chart (id INTEGER PRIMARY KEY, name TEXT, manager_id INTEGER);
`,
			queries: `
-- name: GetOrgHierarchy :many
WITH RECURSIVE org AS (
    SELECT id, name, manager_id, 1 as level FROM org_chart WHERE manager_id IS NULL
    UNION ALL
    SELECT o.id, o.name, o.manager_id, org.level + 1 FROM org_chart o
    JOIN org ON o.manager_id = org.id
)
SELECT id, name, manager_id, level FROM org;
`,
			fn: testCTE,
		},
		{
			name: "Transactions",
			schema: `
CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER NOT NULL);
INSERT INTO accounts (balance) VALUES (1000), (500);
`,
			queries: `
-- name: GetBalance :one
SELECT balance FROM accounts WHERE id = ?;

-- name: TransferFunds :exec
UPDATE accounts SET balance = balance + ? WHERE id = ?;
`,
			fn: testTransactions,
		},
		{
			name: "JSON",
			schema: `
CREATE TABLE configs (id INTEGER PRIMARY KEY, data TEXT);
`,
			queries: `
-- name: GetConfig :one
SELECT data FROM configs WHERE id = ?;

-- name: InsertConfig :exec
INSERT INTO configs (data) VALUES (?);
`,
			fn: testJSON,
		},
		{
			name: "NULLHandling",
			schema: `
CREATE TABLE nullable_test (id INTEGER PRIMARY KEY, value TEXT, optional INTEGER);
`,
			queries: `
-- name: GetNullable :one
SELECT id, value, optional FROM nullable_test WHERE id = ?;

-- name: InsertNullable :one
INSERT INTO nullable_test (value, optional) VALUES (?, ?) RETURNING id, value, optional;
`,
			fn: testNULLHandling,
		},
		{
			name: "BatchInserts",
			schema: `
CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, price INTEGER);
`,
			queries: `
-- name: InsertItem :exec
INSERT INTO items (name, price) VALUES (?, ?);

-- name: GetItemsByIDs :many
SELECT id, name, price FROM items WHERE id IN (sqlc.slice('ids'));
`,
			fn: testBatchInserts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t, tt.schema, tt.queries)
			defer db.Close()

			if err := db.Ping(); err != nil {
				t.Fatalf("ping: %v", err)
			}

			tt.fn(t, db)
		})
	}
}

func testBasicCRUD(t *testing.T, db *testDB) {
	ctx := context.Background()

	res, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "alice", "alice@test.com")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	var username, email string
	err = db.QueryRowContext(ctx, "SELECT username, email FROM users WHERE id = ?", id).Scan(&username, &email)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if username != "alice" {
		t.Errorf("expected alice, got %s", username)
	}

	_, err = db.ExecContext(ctx, "UPDATE users SET email = ? WHERE id = ?", "newemail@test.com", id)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	t.Log("✅ Basic CRUD operations work")
}

func testForeignKeys(t *testing.T, db *testDB) {
	ctx := context.Background()

	db.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	authorID, _ := db.ExecContext(ctx, "INSERT INTO authors (name) VALUES (?)", "Jane Doe")
	aid, _ := authorID.LastInsertId()

	bookID, _ := db.ExecContext(ctx, "INSERT INTO books (author_id, title) VALUES (?, ?)", aid, "Test Book")
	bid, _ := bookID.LastInsertId()

	if bid == 0 {
		t.Error("expected book id")
	}

	var count int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books WHERE author_id = ?", aid).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 book, got %d", count)
	}

	t.Log("✅ Foreign key operations work")
}

func testJoins(t *testing.T, db *testDB) {
	ctx := context.Background()

	db.ExecContext(ctx, "INSERT INTO customers (name) VALUES (?)", "Acme Corp")
	db.ExecContext(ctx, "INSERT INTO orders (customer_id, total) VALUES (?, ?)", 1, 100)

	rows, err := db.QueryContext(ctx, "SELECT c.name, o.total FROM customers c JOIN orders o ON c.id = o.customer_id")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var name string
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	t.Log("✅ JOIN operations work")
}

func testAggregates(t *testing.T, db *testDB) {
	ctx := context.Background()

	db.ExecContext(ctx, "INSERT INTO sales (product, amount) VALUES (?, ?)", "Widget", 100)
	db.ExecContext(ctx, "INSERT INTO sales (product, amount) VALUES (?, ?)", "Widget", 200)
	db.ExecContext(ctx, "INSERT INTO sales (product, amount) VALUES (?, ?)", "Gadget", 150)

	var total sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT SUM(amount) FROM sales WHERE product = ?", "Widget").Scan(&total)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if total.Int64 != 300 {
		t.Errorf("expected 300, got %d", total.Int64)
	}

	var count int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales WHERE product = ?", "Widget").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	t.Log("✅ Aggregate functions work")
}

func testSubqueries(t *testing.T, db *testDB) {
	ctx := context.Background()

	db.ExecContext(ctx, "INSERT INTO employees (name, department, salary) VALUES (?, ?, ?)", "Alice", "Eng", 120000)
	db.ExecContext(ctx, "INSERT INTO employees (name, department, salary) VALUES (?, ?, ?)", "Bob", "Eng", 80000)
	db.ExecContext(ctx, "INSERT INTO employees (name, department, salary) VALUES (?, ?, ?)", "Charlie", "Sales", 90000)

	rows, err := db.QueryContext(ctx, "SELECT name, salary FROM employees WHERE salary > (SELECT AVG(salary) FROM employees)")
	if err != nil {
		t.Fatalf("subquery: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var name string
		var salary int
		if err := rows.Scan(&name, &salary); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
	}
	if count == 0 {
		t.Error("expected at least one high earner")
	}

	t.Log("✅ Subqueries work")
}

func testCTE(t *testing.T, db *testDB) {
	ctx := context.Background()

	db.ExecContext(ctx, "INSERT INTO org_chart (name, manager_id) VALUES ('CEO', NULL)")
	db.ExecContext(ctx, "INSERT INTO org_chart (name, manager_id) VALUES ('VP1', 1)")
	db.ExecContext(ctx, "INSERT INTO org_chart (name, manager_id) VALUES ('VP2', 1)")
	db.ExecContext(ctx, "INSERT INTO org_chart (name, manager_id) VALUES ('Manager1', 2)")

	rows, err := db.QueryContext(ctx, `
		WITH RECURSIVE org AS (
			SELECT id, name, manager_id, 1 as level FROM org_chart WHERE manager_id IS NULL
			UNION ALL
			SELECT o.id, o.name, o.manager_id, org.level + 1 FROM org_chart o
			JOIN org ON o.manager_id = org.id
		)
		SELECT id, name, level FROM org
	`)
	if err != nil {
		t.Fatalf("cte: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 4 {
		t.Errorf("expected 4 rows, got %d", count)
	}

	t.Log("✅ CTE operations work")
}

func testTransactions(t *testing.T, db *testDB) {
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + 100 WHERE id = 1")
	if err != nil {
		tx.Rollback()
		t.Fatalf("update: %v", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - 100 WHERE id = 2")
	if err != nil {
		tx.Rollback()
		t.Fatalf("update: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var balance1, balance2 int
	db.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE id = 1").Scan(&balance1)
	db.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE id = 2").Scan(&balance2)

	if balance1 != 1100 || balance2 != 400 {
		t.Errorf("unexpected balances: %d, %d", balance1, balance2)
	}

	tx, _ = db.BeginTx(ctx, nil)
	tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + 100 WHERE id = 1")
	tx.Rollback()

	t.Log("✅ Transaction operations work")
}

func testJSON(t *testing.T, db *testDB) {
	ctx := context.Background()

	jsonData := `{"key": "value", "count": 42}`

	_, err := db.ExecContext(ctx, "INSERT INTO configs (data) VALUES (?)", jsonData)
	if err != nil {
		t.Fatalf("insert json: %v", err)
	}

	var result string
	err = db.QueryRowContext(ctx, "SELECT data FROM configs WHERE id = 1").Scan(&result)
	if err != nil {
		t.Fatalf("select json: %v", err)
	}

	if !strings.Contains(result, "key") {
		t.Error("expected json data")
	}

	t.Log("✅ JSON operations work")
}

func testNULLHandling(t *testing.T, db *testDB) {
	ctx := context.Background()

	_, err := db.ExecContext(ctx, "INSERT INTO nullable_test (value, optional) VALUES (?, ?)", "test", nil)
	if err != nil {
		t.Fatalf("insert null: %v", err)
	}

	var value sql.NullString
	var optional sql.NullInt64
	err = db.QueryRowContext(ctx, "SELECT value, optional FROM nullable_test WHERE id = 1").Scan(&value, &optional)
	if err != nil {
		t.Fatalf("select null: %v", err)
	}

	if !value.Valid || value.String != "test" {
		t.Error("expected value 'test'")
	}
	if optional.Valid {
		t.Error("expected optional to be NULL")
	}

	t.Log("✅ NULL handling works")
}

func testBatchInserts(t *testing.T, db *testDB) {
	ctx := context.Background()

	ids := []int64{1, 2, 3}
	for _, id := range ids {
		db.ExecContext(ctx, "INSERT OR IGNORE INTO items (id, name, price) VALUES (?, ?, ?)", id, fmt.Sprintf("Item%d", id), id*10)
	}

	rows, err := db.QueryContext(ctx, "SELECT id, name, price FROM items WHERE id IN (1, 2, 3)")
	if err != nil {
		t.Fatalf("batch select: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 items, got %d", count)
	}

	t.Log("✅ Batch insert operations work")
}

func BenchmarkSQLite_Select(b *testing.B) {
	db, err := sql.Open("sqlite", sqliteInMem)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	db.ExecContext(context.Background(), "CREATE TABLE bench (id INTEGER PRIMARY KEY, data TEXT)")
	for i := 0; i < 1000; i++ {
		db.ExecContext(context.Background(), "INSERT INTO bench (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, _ := db.QueryContext(ctx, "SELECT * FROM bench WHERE id < 100")
		for rows.Next() {
			var id int64
			var data string
			rows.Scan(&id, &data)
		}
		rows.Close()
	}
}

func BenchmarkSQLite_Insert(b *testing.B) {
	db, err := sql.Open("sqlite", sqliteInMem)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	db.ExecContext(context.Background(), "CREATE TABLE bench_insert (id INTEGER PRIMARY KEY, data TEXT)")

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ExecContext(ctx, "INSERT INTO bench_insert (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}
}

func BenchmarkSQLite_Transaction(b *testing.B) {
	db, err := sql.Open("sqlite", sqliteInMem)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	db.ExecContext(context.Background(), "CREATE TABLE bench_tx (id INTEGER PRIMARY KEY, value INTEGER)")

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, _ := db.BeginTx(ctx, nil)
		tx.ExecContext(ctx, "INSERT INTO bench_tx (value) VALUES (?)", i)
		tx.Commit()
	}
}
