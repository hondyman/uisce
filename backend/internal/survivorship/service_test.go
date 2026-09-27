package survivorship

import "testing"

func TestNormalizeStrategy(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "SOURCE_PRIORITY", false},
		{"source_priority", "SOURCE_PRIORITY", false},
		{"MOST_RECENT", "MOST_RECENT", false},
		{"bogus", "", true},
	}
	for _, c := range cases {
		got, err := normalizeStrategy(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("normalizeStrategy(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeStrategy(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("normalizeStrategy(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
