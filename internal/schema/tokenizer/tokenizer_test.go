package tokenizer

import (
	"errors"
	"strings"
	"testing"
)

var benchmarkTokens []Token

type tokenExpectation struct {
	kind Kind
	text string
}

func TestScanBasicCreateTable(t *testing.T) {
	sql := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    age INTEGER DEFAULT 42,
    bio TEXT DEFAULT 'unknown',
    payload BLOB DEFAULT X'0A0B'
);
`
	tokens, err := Scan("schema.sql", []byte(sql), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []tokenExpectation{
		{KindKeyword, "CREATE"},
		{KindKeyword, "TABLE"},
		{KindIdentifier, "users"},
		{KindSymbol, "("},
		{KindIdentifier, "id"},
		{KindKeyword, "INTEGER"},
		{KindKeyword, "PRIMARY"},
		{KindKeyword, "KEY"},
		{KindSymbol, ","},
		{KindIdentifier, "name"},
		{KindKeyword, "TEXT"},
		{KindKeyword, "NOT"},
		{KindKeyword, "NULL"},
		{KindSymbol, ","},
		{KindIdentifier, "age"},
		{KindKeyword, "INTEGER"},
		{KindKeyword, "DEFAULT"},
		{KindNumber, "42"},
		{KindSymbol, ","},
		{KindIdentifier, "bio"},
		{KindKeyword, "TEXT"},
		{KindKeyword, "DEFAULT"},
		{KindString, "'unknown'"},
		{KindSymbol, ","},
		{KindIdentifier, "payload"},
		{KindKeyword, "BLOB"},
		{KindKeyword, "DEFAULT"},
		{KindBlob, "X'0A0B'"},
		{KindSymbol, ")"},
		{KindSymbol, ";"},
		{KindEOF, ""},
	}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(expected))
	}
	for i, exp := range expected {
		tok := tokens[i]
		if tok.Kind != exp.kind || tok.Text != exp.text {
			t.Fatalf("token %d mismatch: got (%s,%q), want (%s,%q)", i, tok.Kind, tok.Text, exp.kind, exp.text)
		}
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Fatalf("CREATE position unexpected: got line %d column %d", tokens[0].Line, tokens[0].Column)
	}
	if tokens[2].Line != 1 || tokens[2].Column != 14 {
		t.Fatalf("users position unexpected: got line %d column %d", tokens[2].Line, tokens[2].Column)
	}
	if tokens[4].Line != 2 || tokens[4].Column != 5 {
		t.Fatalf("id position unexpected: got line %d column %d", tokens[4].Line, tokens[4].Column)
	}
}

func TestScanCaptureDocComment(t *testing.T) {
	sql := `-- User table
/* fields: id, name */
CREATE TABLE users (id INTEGER PRIMARY KEY);
`
	tokens, err := Scan("users.sql", []byte(sql), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) == 0 || tokens[0].Kind != KindDocComment {
		t.Fatalf("expected first token to be doc comment, got %v", tokens)
	}
	wantDoc := "User table\nfields: id, name"
	if tokens[0].Text != wantDoc {
		t.Fatalf("doc comment mismatch: got %q want %q", tokens[0].Text, wantDoc)
	}
	if tokens[1].Kind != KindKeyword || tokens[1].Text != "CREATE" {
		t.Fatalf("expected CREATE after doc comment, got %v", tokens[1])
	}
	if tokens[0].File != "users.sql" || tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Fatalf("unexpected doc comment position: %+v", tokens[0])
	}
}

func TestScanCaptureDocCommentDisabled(t *testing.T) {
	sql := `-- User table
CREATE TABLE users (id INTEGER);
`
	tokens, err := Scan("users.sql", []byte(sql), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Kind == KindDocComment {
		t.Fatalf("doc comment should be omitted when captureDocs=false")
	}
	if tokens[0].Kind != KindKeyword || tokens[0].Text != "CREATE" {
		t.Fatalf("expected CREATE as first token, got %v", tokens[0])
	}
}

func TestScanUTF8Identifiers(t *testing.T) {
	sql := "CREATE TABLE café (\"über\" TEXT, `mañana` TEXT, [дата] TEXT);\n"
	tokens, err := Scan("utf8.sql", []byte(sql), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var seenCafe, seenUber, seenManana, seenDate bool
	for _, tok := range tokens {
		switch tok.Text {
		case "café":
			seenCafe = true
		case "\"über\"":
			seenUber = NormalizeIdentifier(tok.Text) == "über"
		case "`mañana`":
			seenManana = NormalizeIdentifier(tok.Text) == "mañana"
		case "[дата]":
			seenDate = NormalizeIdentifier(tok.Text) == "дата"
		}
	}
	if !seenCafe || !seenUber || !seenManana || !seenDate {
		t.Fatalf("missing UTF-8 identifiers: café=%v über=%v mañana=%v дата=%v", seenCafe, seenUber, seenManana, seenDate)
	}
}

func TestScanUnterminatedString(t *testing.T) {
	sql := "CREATE TABLE users (name TEXT DEFAULT 'oops);"
	_, err := Scan("schema.sql", []byte(sql), false)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
	var scanErr *Error
	if !errors.As(err, &scanErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if !strings.Contains(scanErr.Message, "unterminated string") {
		t.Fatalf("unexpected error message: %q", scanErr.Message)
	}
}

func TestScanUnterminatedBlockComment(t *testing.T) {
	sql := "/* missing end"
	_, err := Scan("schema.sql", []byte(sql), false)
	if err == nil {
		t.Fatal("expected error for unterminated block comment")
	}
	var scanErr *Error
	if !errors.As(err, &scanErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if !strings.Contains(scanErr.Message, "unterminated block comment") {
		t.Fatalf("unexpected error message: %q", scanErr.Message)
	}
}

func TestIsKeyword(t *testing.T) {
	testCases := []struct {
		value string
		want  bool
	}{
		{"create", true},
		{"CREATE", true},
		{"integer", true},
		{"unknown", false},
	}
	for _, tc := range testCases {
		if got := IsKeyword(tc.value); got != tc.want {
			t.Fatalf("IsKeyword(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestNormalizeIdentifier(t *testing.T) {
	cases := map[string]string{
		"\"Foo\"":     "Foo",
		"`Bar`":       "Bar",
		"[Baz]":       "Baz",
		"\"Fo\"\"o\"": "Fo\"o",
		"plain":       "plain",
	}
	for input, want := range cases {
		if got := NormalizeIdentifier(input); got != want {
			t.Fatalf("NormalizeIdentifier(%q) = %q, want %q", input, got, want)
		}
	}
}

func BenchmarkScan(b *testing.B) {
	schema := []byte(`CREATE TABLE authors (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE books (
    id INTEGER PRIMARY KEY,
    author_id INTEGER REFERENCES authors(id),
    title TEXT NOT NULL,
    metadata BLOB DEFAULT X'0011'
);
`)
	b.ReportAllocs()
	for b.Loop() {
		toks, err := Scan("bench.sql", schema, true)
		if err != nil {
			b.Fatalf("scan failed: %v", err)
		}
		benchmarkTokens = toks
	}
}
