package model

import (
	"testing"
)

func TestNewCatalog(t *testing.T) {
	c := NewCatalog()

	if c.Tables == nil {
		t.Error("Tables map should be initialized")
	}

	if c.Views == nil {
		t.Error("Views map should be initialized")
	}
}

func TestSortColumns(t *testing.T) {
	cols := []*Column{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "beta"},
	}

	SortColumns(cols)

	if cols[0].Name != "alpha" {
		t.Errorf("first column = %q, want alpha", cols[0].Name)
	}
	if cols[1].Name != "beta" {
		t.Errorf("second column = %q, want beta", cols[1].Name)
	}
	if cols[2].Name != "zebra" {
		t.Errorf("third column = %q, want zebra", cols[2].Name)
	}
}

func TestSortUniqueKeys(t *testing.T) {
	keys := []*UniqueKey{
		{Name: "z", Columns: []string{"a"}},
		{Name: "a", Columns: []string{"z"}},
		{Name: "a", Columns: []string{"a"}},
	}

	SortUniqueKeys(keys)

	// Should be sorted by name, then columns
	if keys[0].Name != "a" || keys[0].Columns[0] != "a" {
		t.Error("first key should be 'a' with column 'a'")
	}
}

func TestSortForeignKeys(t *testing.T) {
	keys := []*ForeignKey{
		{Name: "z", Ref: ForeignKeyRef{Table: "a"}},
		{Name: "a", Ref: ForeignKeyRef{Table: "z"}},
	}

	SortForeignKeys(keys)

	if keys[0].Name != "a" {
		t.Errorf("first key = %q, want a", keys[0].Name)
	}
}

func TestSortIndexes(t *testing.T) {
	idxs := []*Index{
		{Name: "z"},
		{Name: "a"},
		{Name: "m"},
	}

	SortIndexes(idxs)

	if idxs[0].Name != "a" {
		t.Errorf("first index = %q, want a", idxs[0].Name)
	}
	if idxs[2].Name != "z" {
		t.Errorf("last index = %q, want z", idxs[2].Name)
	}
}

func TestValue(t *testing.T) {
	v := &Value{
		Kind: ValueKindString,
		Text: "'hello'",
	}

	if v.Kind != ValueKindString {
		t.Errorf("Kind = %v, want ValueKindString", v.Kind)
	}

	if v.Text != "'hello'" {
		t.Errorf("Text = %q, want \"'hello'\"", v.Text)
	}
}

func TestValueKinds(t *testing.T) {
	tests := []struct {
		kind ValueKind
		name string
	}{
		{ValueKindUnknown, "Unknown"},
		{ValueKindNumber, "Number"},
		{ValueKindString, "String"},
		{ValueKindBlob, "Blob"},
		{ValueKindKeyword, "Keyword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Value{Kind: tt.kind}
			if v.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", v.Kind, tt.kind)
			}
		})
	}
}

func TestTable(t *testing.T) {
	table := &Table{
		Name:         "users",
		Doc:          "User table",
		WithoutRowID: true,
		Strict:       true,
		Columns: []*Column{
			{Name: "id", Type: "INTEGER", NotNull: true},
			{Name: "name", Type: "TEXT"},
		},
	}

	if table.Name != "users" {
		t.Errorf("Name = %q, want users", table.Name)
	}
	if table.Doc != "User table" {
		t.Errorf("Doc = %q, want 'User table'", table.Doc)
	}
	if !table.WithoutRowID {
		t.Error("WithoutRowID should be true")
	}
	if !table.Strict {
		t.Error("Strict should be true")
	}
	if len(table.Columns) != 2 {
		t.Errorf("len(Columns) = %d, want 2", len(table.Columns))
	}
}

func TestColumn(t *testing.T) {
	col := &Column{
		Name:    "id",
		Type:    "INTEGER",
		NotNull: true,
	}

	if col.Name != "id" {
		t.Errorf("Name = %q, want id", col.Name)
	}
	if col.Type != "INTEGER" {
		t.Errorf("Type = %q, want INTEGER", col.Type)
	}
	if !col.NotNull {
		t.Error("NotNull should be true")
	}
}

func TestView(t *testing.T) {
	view := &View{
		Name: "active_users",
		Doc:  "Active users view",
		SQL:  "SELECT * FROM users WHERE active = 1",
	}

	if view.Name != "active_users" {
		t.Errorf("Name = %q, want active_users", view.Name)
	}
	if view.Doc != "Active users view" {
		t.Errorf("Doc = %q, want 'Active users view'", view.Doc)
	}
	if view.SQL != "SELECT * FROM users WHERE active = 1" {
		t.Errorf("SQL = %q, want 'SELECT * FROM users WHERE active = 1'", view.SQL)
	}
}

func TestPrimaryKey(t *testing.T) {
	pk := &PrimaryKey{
		Name:    "pk_users",
		Columns: []string{"id"},
	}

	if pk.Name != "pk_users" {
		t.Errorf("Name = %q, want pk_users", pk.Name)
	}
	if len(pk.Columns) != 1 || pk.Columns[0] != "id" {
		t.Errorf("Columns = %v, want [id]", pk.Columns)
	}
}

func TestForeignKey(t *testing.T) {
	fk := &ForeignKey{
		Name:    "fk_user_orders",
		Columns: []string{"user_id"},
		Ref: ForeignKeyRef{
			Table:   "users",
			Columns: []string{"id"},
		},
	}

	if fk.Name != "fk_user_orders" {
		t.Errorf("Name = %q, want fk_user_orders", fk.Name)
	}
	if fk.Ref.Table != "users" {
		t.Errorf("Ref.Table = %q, want users", fk.Ref.Table)
	}
}

func TestIndex(t *testing.T) {
	idx := &Index{
		Name:    "idx_users_email",
		Unique:  true,
		Columns: []string{"email"},
	}

	if idx.Name != "idx_users_email" {
		t.Errorf("Name = %q, want idx_users_email", idx.Name)
	}
	if !idx.Unique {
		t.Error("Unique should be true")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Columns = %v, want [email]", idx.Columns)
	}
}
