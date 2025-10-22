package sqlfix

import "testing"

func TestAliasGenerator_Aggregates(t *testing.T) {
	g := NewAliasGenerator()
	alias := g.Next("COUNT(*)")
	if alias != "count_all" {
		t.Fatalf("expected count_all, got %s", alias)
	}

	alias = g.Next("SUM(payments.amount)")
	if alias != "sum_amount" {
		t.Fatalf("expected sum_amount, got %s", alias)
	}
}

func TestAliasGenerator_Literals(t *testing.T) {
	g := NewAliasGenerator()
	cases := map[string]string{
		"TRUE":      "flag_true",
		"FALSE":     "flag_false",
		"42":        "const_42",
		"'pending'": "const_pending",
	}
	for expr, want := range cases {
		if got := g.Next(expr); got != want {
			t.Fatalf("%s: expected %s, got %s", expr, want, got)
		}
	}
}

func TestAliasGenerator_Expressions(t *testing.T) {
	g := NewAliasGenerator()
	if got := g.Next("balance - tax"); got != "balance_minus_tax" {
		t.Fatalf("expected balance_minus_tax, got %s", got)
	}
	if got := g.Next("balance - tax"); got != "balance_minus_tax_2" {
		t.Fatalf("expected balance_minus_tax_2, got %s", got)
	}
}

func TestAliasGenerator_Reserve(t *testing.T) {
	g := NewAliasGenerator()
	g.Reserve("count_all")
	if got := g.Next("COUNT(*)"); got != "count_all_2" {
		t.Fatalf("expected count_all_2, got %s", got)
	}
}
