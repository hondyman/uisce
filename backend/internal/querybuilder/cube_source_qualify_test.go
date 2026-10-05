package querybuilder

import "testing"

func TestQualifyCubeSourceTable(t *testing.T) {
	t.Setenv("CUBE_SOURCE_CATALOG", "pg_alpha")

	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"/orm/account", "pg_alpha.orm.account"},
		{"/oms/position", "pg_alpha.oms.position"},
		{"oms.security", "oms.security"},
		{"pg_alpha.oms.account", "pg_alpha.oms.account"},
		{"  /mdm/security_master  ", "pg_alpha.mdm.security_master"},
	}
	for _, c := range cases {
		got := QualifyCubeSourceTable(c.in)
		if got != c.want {
			t.Errorf("QualifyCubeSourceTable(%q)=%q want %q", c.in, got, c.want)
		}
	}

	t.Setenv("CUBE_SOURCE_CATALOG", "lake_pg")
	if got := QualifyCubeSourceTable("/orm/account"); got != "lake_pg.orm.account" {
		t.Errorf("custom catalog: got %q", got)
	}
}
