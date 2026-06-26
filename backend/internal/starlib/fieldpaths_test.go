package starlib

import "testing"

func TestExtractRequiredFieldPaths(t *testing.T) {
	script := `
ok = and_(
  eq(field("account", "account_type"), "ADVISORY"),
  gt(num_field("account", "aum"), 100),
)
_ = F("household.household_id")
_ = F("account", "id")
# dynamic should be ignored
k = "foo"
_ = field("account", k)
`

	paths, err := ExtractRequiredFieldPaths(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expect := map[string]bool{
		"account.account_type":   true,
		"account.aum":            true,
		"household.household_id": true,
		"account.id":             true,
	}

	if len(paths) != len(expect) {
		t.Fatalf("len(paths)=%d want %d: %#v", len(paths), len(expect), paths)
	}
	for _, p := range paths {
		if !expect[p] {
			t.Fatalf("unexpected path: %q (all=%#v)", p, paths)
		}
	}
}
