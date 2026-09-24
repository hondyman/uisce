package metadata

import (
	"reflect"
	"testing"
)

func TestParseSchemaWhitelist(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"orm", []string{"orm"}},
		{"orm,vend,ref,mdm,cash_flow,wlth", []string{"orm", "vend", "ref", "mdm", "cash_flow", "wlth"}},
		{" orm , mdm ", []string{"orm", "mdm"}},
		{"orm,,mdm,", []string{"orm", "mdm"}},
		{"orm,mdm,orm", []string{"orm", "mdm"}},
	}
	for _, c := range cases {
		if got := parseSchemaWhitelist(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseSchemaWhitelist(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}
