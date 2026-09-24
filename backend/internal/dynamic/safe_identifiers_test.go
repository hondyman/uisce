package dynamic

import (
	"testing"
)

func TestSafeIdentifier(t *testing.T) {
	cases := []struct {
		in   string
		want bool // true = pass (nil error), false = reject
	}{
		// Valid simple identifiers (lowercase only per identifierPattern)
		{"status", true},
		{"subtype_code", true},
		{"base_currency", true},
		{"my_column", true},
		{"a", true},
		{"_private", true},
		// Valid qualified identifiers
		{"oms.account", true},
		{"master.customer", true},
		{"schema.table", true},
		// Invalid: uppercase letters are NOT in the pattern
		{"STATUS", false},
		{"OMS", false},
		{"OMS.Account", false},
		// Invalid: leading digit
		{"1column", false},
		{"9table", false},
		// Invalid: dot-only (must have identifier on each side)
		{".table", false},
		{"table.", false},
		{"..", false},
		// Invalid: empty
		{"", false},
		// Invalid: space
		{"table name", false},
		// Invalid: SQL injection attempts
		{"table; DROP TABLE", false},
		{"table--comment", false},
		{"1=1", false},
		{"'; DROP TABLE--", false},
		{"oms.account UNION SELECT", false},
		// Invalid: backtick, parens
		{"`status`", false},
		{"status)", false},
		{"(status", false},
	}

	for _, c := range cases {
		err := SafeIdentifier(c.in)
		got := (err == nil)
		if got != c.want {
			t.Errorf("SafeIdentifier(%q): got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSafeAlias(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"a", true},
		{"t1", true},
		{"alias_name", true},
		{"_priv", true},
		// Dot is NOT allowed in aliases
		{"a.b", false},
		{"schema.t1", false},
		// Empty
		{"", false},
		// Leading digit
		{"1alias", false},
		// SQL injection
		{"alias; DROP", false},
		{"alias--", false},
	}

	for _, c := range cases {
		err := SafeAlias(c.in)
		got := (err == nil)
		if got != c.want {
			t.Errorf("SafeAlias(%q): got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNormalizeAndValidateSource(t *testing.T) {
	cases := []struct {
		table    string
		column   string
		wantPass bool
	}{
		// Valid: real tables and columns (case-insensitive)
		{"oms.account", "status", true},
		{"OMS.ACCOUNT", "STATUS", true},
		{"Oms.Account", "Status", true},
		{"oms.trade_order", "order_side", true},
		{"oms.trade_order", "order_status", true},
		{"oms.position", "subtype_code", true},
		{"master.customer", "kyc_status", true},
		{"master.sales_ledger", "invoice_status", true},
		// Invalid: unknown table
		{"users", "status", false},
		{"public.orders", "status", false},
		{"oms.account", "email", false},          // column not in allow-list for this table
		{"oms.trade_order", "sponsor_id", false}, // not a column for trade_order
		{"oms.position", "sponsor_id", false},
		// Invalid: empty
		{"", "status", false},
		{"oms.account", "", false},
		// SQL injection attempts
		{"oms.account; DROP TABLE x", "status", false},
		{"oms.account", "status UNION SELECT", false},
	}

	for _, c := range cases {
		_, _, err := NormalizeAndValidateSource(c.table, c.column)
		got := (err == nil)
		if got != c.wantPass {
			t.Errorf("NormalizeAndValidateSource(%q, %q): got %v, want %v", c.table, c.column, got, c.wantPass)
		}
	}
}

func TestValidateTable(t *testing.T) {
	cases := []struct {
		table    string
		wantPass bool
	}{
		{"oms.account", true},
		{"OMS.ACCOUNT", true},
		{"oms.trade_order", true},
		{"oms.position", true},
		{"master.customer", true},
		{"master.sales_ledger", true},
		// Unknown table
		{"users", false},
		{"public.orders", false},
		{"unknown.table", false},
		// SQL injection
		{"oms.account; DROP TABLE", false},
		{"oms.account--", false},
		// Empty
		{"", false},
	}

	for _, c := range cases {
		_, err := ValidateTable(c.table)
		got := (err == nil)
		if got != c.wantPass {
			t.Errorf("ValidateTable(%q): got %v, want %v", c.table, got, c.wantPass)
		}
	}
}

func TestNormalizeAndValidateAlias(t *testing.T) {
	cases := []struct {
		alias string
		want  string // normalized value; empty string means error expected
		ok    bool   // true = expect no error
	}{
		{"t1", "t1", true},
		{"alias", "alias", true},
		{"ALIAS", "alias", true}, // lowercased
		{"Alias", "alias", true},
		// Empty is valid — caller falls back to generated alias
		{"", "", true},
		// Dot not allowed
		{"a.b", "", false},
		// SQL injection
		{"alias; DROP", "", false},
		{"--comment", "", false},
	}

	for _, c := range cases {
		got, err := NormalizeAndValidateAlias(c.alias)
		if !c.ok {
			if err == nil {
				t.Errorf("NormalizeAndValidateAlias(%q): expected error, got nil", c.alias)
			}
		} else {
			if err != nil {
				t.Errorf("NormalizeAndValidateAlias(%q): unexpected error: %v", c.alias, err)
			} else if got != c.want {
				t.Errorf("NormalizeAndValidateAlias(%q): got %q, want %q", c.alias, got, c.want)
			}
		}
	}
}

func TestInSet(t *testing.T) {
	tzAllowed := map[string]bool{
		"UTC":              true,
		"America/New_York": true,
		"America/Chicago":  true,
		"Europe/London":    true,
	}

	cases := []struct {
		value string
		label string
		set   map[string]bool
		want  bool
	}{
		// Case-significant: must match exactly
		{"UTC", "timezone", tzAllowed, true},
		{"utc", "timezone", tzAllowed, false}, // lowercase rejected
		{"America/New_York", "timezone", tzAllowed, true},
		{"utc", "timezone", tzAllowed, false},
		{"Europe/London", "timezone", tzAllowed, true},
		{"Europe/london", "timezone", tzAllowed, false},
		{"GMT", "timezone", tzAllowed, false},
		{"", "timezone", tzAllowed, false},
	}

	for _, c := range cases {
		err := InSet(c.value, c.label, c.set)
		got := (err == nil)
		if got != c.want {
			t.Errorf("InSet(%q, %q, ...): got %v, want %v", c.value, c.label, got, c.want)
		}
	}
}

func TestSafeFormatString(t *testing.T) {
	cases := []struct {
		in   string
		want bool // true = accept, false = reject
		desc string
	}{
		// Valid: empty is valid (optional field may be omitted)
		{"", true, "empty string is valid"},

		// Valid: common Postgres format strings
		{"YYYY-MM-DD", true, "ISO date"},
		{"yyyy-mm-dd", true, "lowercase equivalent"},
		{"YYYY-MM-DD HH24:MI:SS", true, "datetime with time"},
		{"YYYY-MM-DD HH24:MI:SS.US", true, "datetime with microseconds"},
		{"YYYY/MM/DD", true, "slash separator"},
		{"DD.MM.YYYY", true, "European dot separator"},
		{"HH24:MI", true, "time only"},
		{"YYYY", true, "year only"},
		{"MM", true, "month only"},
		{"DD", true, "day only"},
		{"HH12:MI AM", true, "12-hour with AM/PM"},
		{"YYYY-MM-DD TZH:TZM", true, "timezone offset"},

		// Invalid: Oracle-only format elements (FF1-FF6 don't exist in Postgres).
		// Tokenizer splits FF1 into "FF" + "1"; both are rejected.
		{"FF1", false, "Oracle fractional seconds"},
		{"FF2", false, "Oracle fractional seconds"},
		{"FF6", false, "Oracle fractional seconds"},
		{"YYYYFF1", false, "format element with FF"},
		{"FF1YYYY", false, "FF at start"},

		// Invalid: quote-breakout attempts
		{"YYYY'-'MM'-'DD", false, "literal quote in format"},
		{"YYYY'); DROP TABLE--", false, "SQL injection in format"},
		{"'); DROP TABLE--", false, "quote breakout start"},
		{"YYYY'\\'MM", false, "backslash in format"},
		{`YYYY\'MM`, false, "escaped quote in format"},

		// Invalid: unsupported separators (not in allowedSeparatorRunes)
		{"YYYY|MM|DD", false, "pipe separator not allowed"},
		{"YYYY#MM#DD", false, "hash separator not allowed"},
		{"YYYY_MM_DD", false, "underscore not allowed"},

		// Invalid: unsupported format elements
		{"INVALID", false, "unknown format element"},
		{"XXXX", false, "not a format element"},
	}

	for _, c := range cases {
		err := SafeFormatString(c.in)
		got := (err == nil)
		if got != c.want {
			t.Errorf("SafeFormatString(%q): got %v, want %v (%s)", c.in, got, c.want, c.desc)
		}
	}
}

func TestSafeLiteral(t *testing.T) {
	cases := []struct {
		in   string
		want bool
		desc string
	}{
		// Valid: plain alphanumerics and conservative separators
		{"2024-01-15", true, "date-like literal"},
		{"fallback_value", true, "simple fallback"},
		{"N/A", true, "N/A fallback"},
		{"unknown", true, "simple string"},
		{"00", true, "numeric string"},
		{"default", true, "default literal"},
		{"2024/01/15", true, "slash date"},
		{"10:30:00", true, "time string"},
		{"John Doe", true, "name with space"},

		// Invalid: quote-breakout attempts
		{"'; DROP TABLE--", false, "SQL injection quote breakout"},
		{"O'Reilly", false, "single quote in name"},
		{"it''s", false, "double single quote"},
		{"line1\nline2", false, "newline in literal"},
		{"null\x00byte", false, "NUL byte"},
		{"path\\to\\file", false, "backslash in path"},
		{"C:\\Windows", false, "Windows path with backslash"},
	}

	for _, c := range cases {
		err := SafeLiteral(c.in)
		got := (err == nil)
		if got != c.want {
			t.Errorf("SafeLiteral(%q): got %v, want %v (%s)", c.in, got, c.want, c.desc)
		}
	}
}

// TestAllowListMapsSanity verifies the allow-list maps contain expected entries.
// These are the ground truth for the validation functions.
func TestAllowListMapsSanity(t *testing.T) {
	// AllowedEnumSources should have the 5 real STI tables
	wantTables := []string{"oms.account", "oms.trade_order", "oms.position", "master.customer", "master.sales_ledger"}
	for _, tbl := range wantTables {
		if cols, ok := AllowedEnumSources[tbl]; !ok {
			t.Errorf("AllowedEnumSources missing expected table %q", tbl)
		} else if len(cols) == 0 {
			t.Errorf("AllowedEnumSources[%q] has no columns", tbl)
		}
	}

	// AllowedDateFormatTokens should NOT contain Oracle-only elements
	oracleOnly := []string{"FF1", "FF2", "FF3", "FF4", "FF5", "FF6"}
	for _, tok := range oracleOnly {
		if AllowedDateFormatTokens[tok] {
			t.Errorf("AllowedDateFormatTokens contains Oracle-only element %q", tok)
		}
	}

	// AllowedDateFormatTokens SHOULD contain real Postgres elements
	postgresElements := []string{"YYYY", "YY", "MM", "DD", "HH24", "MI", "SS", "MS", "US", "TZH", "TZM"}
	for _, tok := range postgresElements {
		if !AllowedDateFormatTokens[tok] {
			t.Errorf("AllowedDateFormatTokens missing Postgres element %q", tok)
		}
	}

	// AllowedEnumParsingFunction should NOT contain invented functions
	if AllowedEnumParsingFunction["PARSE_TIMESTAMP"] {
		t.Errorf("AllowedEnumParsingFunction contains invented PARSE_TIMESTAMP")
	}
	if !AllowedEnumParsingFunction["TO_TIMESTAMP"] {
		t.Errorf("AllowedEnumParsingFunction missing real function TO_TIMESTAMP")
	}

	// AllowedUnionTypes should contain expected set operations
	wantUnions := []string{"UNION", "UNION ALL", "INTERSECT", "INTERSECT ALL", "EXCEPT", "EXCEPT ALL"}
	for _, u := range wantUnions {
		if !AllowedUnionTypes[u] {
			t.Errorf("AllowedUnionTypes missing %q", u)
		}
	}
}
