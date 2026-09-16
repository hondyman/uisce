package trading

import "testing"

func TestSwapDBName(t *testing.T) {
	got := swapDBName("postgresql://u:p@127.0.0.1:5432/alpha?sslmode=disable", "crims")
	if got != "postgresql://u:p@127.0.0.1:5432/crims?sslmode=disable" {
		t.Fatalf("got %q", got)
	}
}
