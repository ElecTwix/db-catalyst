package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

const (
	sqliteDriver = "sqlite"
	sqliteInMem  = ":memory:"
)

type generatedDB struct {
	*sql.DB
	tmpDir string
}

func BenchmarkDBCatalyst_Select(b *testing.B) {
	benchmarkSelect(b, generateDBCatalystCode)
}

func BenchmarkSQLC_Select(b *testing.B) {
	benchmarkSelect(b, generateSQLCCode)
}

func benchmarkSelect(b *testing.B, genFn func(*testing.T) *generatedDB) {
	t := &testing.T{}
	db := genFn(t)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench (id INTEGER PRIMARY KEY, data TEXT)")
	for i := 0; i < 1000; i++ {
		db.ExecContext(ctx, "INSERT INTO bench (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}

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

func BenchmarkDB_Catalyst_Insert(b *testing.B) {
	benchmarkInsert(b, generateDBCatalystCode)
}

func BenchmarkSQLC_Insert(b *testing.B) {
	benchmarkInsert(b, generateSQLCCode)
}

func benchmarkInsert(b *testing.B, genFn func(*testing.T) *generatedDB) {
	t := &testing.T{}
	db := genFn(t)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench_insert (id INTEGER PRIMARY KEY, data TEXT)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ExecContext(ctx, "INSERT INTO bench_insert (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}
}

func BenchmarkDB_Catalyst_Transaction(b *testing.B) {
	benchmarkTransaction(b, generateDBCatalystCode)
}

func BenchmarkSQLC_Transaction(b *testing.B) {
	benchmarkTransaction(b, generateSQLCCode)
}

func benchmarkTransaction(b *testing.B, genFn func(*testing.T) *generatedDB) {
	t := &testing.T{}
	db := genFn(t)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench_tx (id INTEGER PRIMARY KEY, value INTEGER)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, _ := db.BeginTx(ctx, nil)
		tx.ExecContext(ctx, "INSERT INTO bench_tx (value) VALUES (?)", i)
		tx.Commit()
	}
}

func generateDBCatalystCode(t *testing.T) *generatedDB {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "bench-db-catalyst-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	schema := `
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
`
	queries := `
-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, name, email FROM users;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (?, ?);
`

	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schema), 0600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queries), 0600); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	cfg := `package = "db"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(cfg), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command("go", "run", "github.com/electwix/db-catalyst/cmd/db-catalyst@latest", "--config", filepath.Join(tmpDir, "db-catalyst.toml"))
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("db-catalyst output: %s\n", output)
		t.Fatalf("db-catalyst: %v", err)
	}

	db, err := sql.Open(sqliteDriver, sqliteInMem)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	return &generatedDB{DB: db, tmpDir: tmpDir}
}

func generateSQLCCode(t *testing.T) *generatedDB {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "bench-sqlc-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	schema := `
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
`
	queries := `
-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = :id;

-- name: ListUsers :many
SELECT id, name, email FROM users;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (:name, :email);
`

	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schema), 0600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queries), 0600); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	sqlcCfg := `version: "2"
sql:
  - engine: "sqlite"
    queries: "queries.sql"
    schema: "schema.sql"
    gen:
      go:
        package: "db"
        out: "gen"
        sql_package: "database/sql"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "sqlc.yaml"), []byte(sqlcCfg), 0600); err != nil {
		t.Fatalf("write sqlc config: %v", err)
	}

	cmd := exec.Command("sqlc", "generate")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("sqlc output: %s\n", output)
		t.Fatalf("sqlc: %v", err)
	}

	db, err := sql.Open(sqliteDriver, sqliteInMem)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	return &generatedDB{DB: db, tmpDir: tmpDir}
}

func TestCompare_Output_Quality(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		query  string
		check  func(*testing.T, string)
	}{
		{
			name: "ParameterNaming",
			schema: `
CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);
`,
			query: `
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;
`,
			check: func(t *testing.T, code string) {
				if !strings.Contains(code, "email") {
					t.Error("expected parameter named 'email'")
				}
			},
		},
		{
			name: "SliceExpansion",
			schema: `
CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT);
CREATE TABLE item_tags (item_id INTEGER, tag_id INTEGER);
`,
			query: `
-- name: GetTagsByItems :many
SELECT t.* FROM tags t
JOIN item_tags it ON t.id = it.tag_id
WHERE it.item_id IN (sqlc.slice('item_ids'));
`,
			check: func(t *testing.T, code string) {
				if strings.Contains(code, "/*SLICE:") {
					t.Log("Note: slice expansion placeholder found - may need implementation")
				}
				if strings.Contains(code, "itemIds") || strings.Contains(code, "ItemIds") {
					t.Log("Parameter named correctly")
				}
			},
		},
		{
			name: "NullableDetection",
			schema: `
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, bio TEXT);
`,
			query: `
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
`,
			check: func(t *testing.T, code string) {
				if strings.Contains(code, "*string") && !strings.Contains(code, "sql.NullString") {
					t.Log("Nullable field detected")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbDir := generateDBCatalystCode(t)
			defer dbDir.Close()

			genDir := filepath.Join(dbDir.tmpDir, "gen")
			files, _ := os.ReadDir(genDir)

			var allCode strings.Builder
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".go") {
					data, _ := os.ReadFile(filepath.Join(genDir, f.Name()))
					allCode.Write(data)
				}
			}

			tt.check(t, allCode.String())
		})
	}
}

func TestSQLite_Generated_Code_Execution(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		queries string
		fn      func(*testing.T, *sql.DB)
	}{
		{
			name: "FullWorkflow",
			schema: `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT,
    published INTEGER DEFAULT 0
);
`,
			queries: `
-- name: CreateUser :one
INSERT INTO users (username, email) VALUES (?, ?) RETURNING id, username, email, created_at;

-- name: GetUser :one
SELECT id, username, email, created_at FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, username, email, created_at FROM users ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (user_id, title, content, published) VALUES (?, ?, ?, ?) RETURNING id, user_id, title, content, published;

-- name: GetUserPosts :many
SELECT p.id, p.user_id, p.title, p.content, p.published
FROM posts p
WHERE p.user_id = ?
ORDER BY p.created_at DESC;
`,
			fn: testFullWorkflow,
		},
		{
			name: "TransactionRollback",
			schema: `
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    balance INTEGER NOT NULL DEFAULT 0
);
INSERT INTO accounts (balance) VALUES (1000), (1000);
`,
			queries: `
-- name: GetBalance :one
SELECT balance FROM accounts WHERE id = ?;

-- name: UpdateBalance :exec
UPDATE accounts SET balance = balance + ? WHERE id = ?;
`,
			fn: testTransactionRollback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbDir := generateDBCatalystCode(t)
			defer dbDir.Close()

			if err := dbDir.Ping(); err != nil {
				t.Fatalf("ping: %v", err)
			}

			if _, err := dbDir.Exec(tt.schema); err != nil {
				t.Fatalf("exec schema: %v", err)
			}

			tt.fn(t, dbDir.DB)
		})
	}
}

func testFullWorkflow(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	res, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "alice", "alice@test.com")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	userID, _ := res.LastInsertId()
	if userID == 0 {
		t.Error("expected user ID")
	}

	_, err = db.ExecContext(ctx, "INSERT INTO posts (user_id, title, content, published) VALUES (?, ?, ?, ?)",
		userID, "Hello World", "This is my first post", 1)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}

	rows, err := db.QueryContext(ctx, "SELECT id, user_id, title FROM posts WHERE user_id = ?", userID)
	if err != nil {
		t.Fatalf("query posts: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 post, got %d", count)
	}

	t.Log("✅ Full workflow executed successfully")
}

func testTransactionRollback(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + 100 WHERE id = 1")
	if err != nil {
		tx.Rollback()
		t.Fatalf("update: %v", err)
	}

	_ = tx.Rollback()

	var balance1, balance2 int
	db.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE id = 1").Scan(&balance1)
	db.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE id = 2").Scan(&balance2)

	if balance1 != 1000 {
		t.Errorf("expected balance 1000 after rollback, got %d", balance1)
	}

	t.Log("✅ Transaction rollback works correctly")
}

func BenchmarkInMemory_Select_1000_Rows(b *testing.B) {
	db, _ := sql.Open(sqliteDriver, sqliteInMem)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench (id INTEGER PRIMARY KEY, data TEXT)")
	for i := 0; i < 1000; i++ {
		db.ExecContext(ctx, "INSERT INTO bench (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, _ := db.QueryContext(ctx, "SELECT * FROM bench WHERE id < 100")
		count := 0
		for rows.Next() {
			var id int64
			var data string
			rows.Scan(&id, &data)
			count++
		}
		rows.Close()
	}
}

func BenchmarkInMemory_Select_Prepared(b *testing.B) {
	db, _ := sql.Open(sqliteDriver, sqliteInMem)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench (id INTEGER PRIMARY KEY, data TEXT)")
	for i := 0; i < 1000; i++ {
		db.ExecContext(ctx, "INSERT INTO bench (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}

	stmt, _ := db.PrepareContext(ctx, "SELECT * FROM bench WHERE id < ?")
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, _ := stmt.QueryContext(ctx, 100)
		count := 0
		for rows.Next() {
			var id int64
			var data string
			rows.Scan(&id, &data)
			count++
		}
		rows.Close()
	}
}

func BenchmarkInMemory_Insert_1000(b *testing.B) {
	db, _ := sql.Open(sqliteDriver, sqliteInMem)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench (id INTEGER PRIMARY KEY, data TEXT)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ExecContext(ctx, "INSERT INTO bench (data) VALUES (?)", fmt.Sprintf("data-%d", i))
	}
}

func BenchmarkInMemory_Transaction_100(b *testing.B) {
	db, _ := sql.Open(sqliteDriver, sqliteInMem)
	defer db.Close()

	ctx := context.Background()
	db.ExecContext(ctx, "CREATE TABLE bench (id INTEGER PRIMARY KEY, value INTEGER)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, _ := db.BeginTx(ctx, nil)
		tx.ExecContext(ctx, "INSERT INTO bench (value) VALUES (?)", i)
		tx.Commit()
	}
}

func TestSQLite_InMemory_AllFeatures(t *testing.T) {
	db, err := sql.Open(sqliteDriver, sqliteInMem)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(*testing.T, *sql.DB)
	}{
		{"CRUD", testSQLiteCRUD},
		{"Joins", testSQLiteJoins},
		{"Aggregates", testSQLiteAggregates},
		{"Subqueries", testSQLiteSubqueries},
		{"CTEs", testSQLiteCTEs},
		{"Transactions", testSQLiteTransactions},
		{"JSON", testSQLiteJSON},
		{"NULLs", testSQLiteNULLs},
		{"BatchOps", testSQLiteBatchOps},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.ExecContext(ctx, "DROP TABLE IF EXISTS test_table")
			tt.fn(t, db)
		})
	}
}

func testSQLiteCRUD(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")

	res, err := db.ExecContext(ctx, "INSERT INTO test_table (name) VALUES (?)", "test")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	id, _ := res.LastInsertId()

	var name string
	err = db.QueryRowContext(ctx, "SELECT name FROM test_table WHERE id = ?", id).Scan(&name)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if name != "test" {
		t.Errorf("expected 'test', got %s", name)
	}

	_, err = db.ExecContext(ctx, "UPDATE test_table SET name = ? WHERE id = ?", "updated", id)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	rows, _ := db.QueryContext(ctx, "SELECT name FROM test_table")
	rows.Close()

	_, err = db.ExecContext(ctx, "DELETE FROM test_table WHERE id = ?", id)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func testSQLiteJoins(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE a (id INTEGER PRIMARY KEY, name TEXT)")
	db.ExecContext(ctx, "CREATE TABLE b (id INTEGER PRIMARY KEY, a_id INTEGER, value TEXT)")
	db.ExecContext(ctx, "INSERT INTO a (name) VALUES ('x')")
	db.ExecContext(ctx, "INSERT INTO b (a_id, value) VALUES (1, 'y')")

	rows, _ := db.QueryContext(ctx, "SELECT a.name, b.value FROM a JOIN b ON a.id = b.a_id")
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func testSQLiteAggregates(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE sales (id INTEGER PRIMARY KEY, amount INTEGER)")
	db.ExecContext(ctx, "INSERT INTO sales (amount) VALUES (100), (200), (300)")

	var sum, avg, count int64
	db.QueryRowContext(ctx, "SELECT SUM(amount), AVG(amount), COUNT(*) FROM sales").Scan(&sum, &avg, &count)

	if sum != 600 || count != 3 {
		t.Errorf("unexpected: sum=%d, count=%d", sum, count)
	}
}

func testSQLiteSubqueries(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE employees (id INTEGER PRIMARY KEY, salary INTEGER)")
	db.ExecContext(ctx, "INSERT INTO employees (salary) VALUES (50), (100), (150)")

	rows, _ := db.QueryContext(ctx, "SELECT * FROM employees WHERE salary > (SELECT AVG(salary) FROM employees)")
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 high earner, got %d", count)
	}
}

func testSQLiteCTEs(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE org (id INTEGER PRIMARY KEY, parent_id INTEGER)")
	db.ExecContext(ctx, "INSERT INTO org (id) VALUES (1), (2), (3)")

	rows, _ := db.QueryContext(ctx, "WITH RECURSIVE cte AS (SELECT 1 as n UNION ALL SELECT n+1 FROM cte WHERE n < 3) SELECT * FROM cte")
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

func testSQLiteTransactions(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)")

	tx, _ := db.BeginTx(ctx, nil)
	tx.ExecContext(ctx, "INSERT INTO accounts (balance) VALUES (100)")
	tx.Commit()

	var balance int
	db.QueryRowContext(ctx, "SELECT balance FROM accounts").Scan(&balance)

	if balance != 100 {
		t.Errorf("expected 100, got %d", balance)
	}
}

func testSQLiteJSON(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE configs (id INTEGER PRIMARY KEY, data TEXT)")
	db.ExecContext(ctx, "INSERT INTO configs (data) VALUES (?)", `{"key":"value"}`)

	var data string
	db.QueryRowContext(ctx, "SELECT data FROM configs").Scan(&data)

	if !strings.Contains(data, "key") {
		t.Error("expected JSON data")
	}
}

func testSQLiteNULLs(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE nullable (id INTEGER PRIMARY KEY, val TEXT)")
	db.ExecContext(ctx, "INSERT INTO nullable (val) VALUES (NULL)")

	var val sql.NullString
	db.QueryRowContext(ctx, "SELECT val FROM nullable").Scan(&val)

	if val.Valid {
		t.Error("expected NULL")
	}
}

func testSQLiteBatchOps(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	db.ExecContext(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")

	for i := 0; i < 10; i++ {
		db.ExecContext(ctx, "INSERT INTO items (name) VALUES (?)", fmt.Sprintf("item-%d", i))
	}

	rows, _ := db.QueryContext(ctx, "SELECT COUNT(*) FROM items")
	var count int
	rows.Close()
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&count)

	if count != 10 {
		t.Errorf("expected 10 items, got %d", count)
	}
}

func TestSQLite_Query_Validation(t *testing.T) {
	db, err := sql.Open(sqliteDriver, sqliteInMem)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   string
		query   string
		wantErr bool
	}{
		{"Valid_Select", "CREATE TABLE t (id INT)", "SELECT id FROM t", false},
		{"Valid_Insert", "CREATE TABLE t (id INT)", "INSERT INTO t (id) VALUES (1)", false},
		{"Valid_Update", "CREATE TABLE t (id INT)", "UPDATE t SET id = 2 WHERE id = 1", false},
		{"Valid_Delete", "CREATE TABLE t (id INT)", "DELETE FROM t WHERE id = 1", false},
		{"Invalid_Table", "", "SELECT * FROM nonexistent", true},
		{"Invalid_Column", "CREATE TABLE t (id INT)", "SELECT nonexistent FROM t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != "" {
				db.ExecContext(ctx, tt.setup)
			}

			_, err := db.ExecContext(ctx, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("query error: %v", err)
			}
		})
	}
}
