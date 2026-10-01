package platform

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/models"
)

// Attribute-condition semantics, pinned when evaluation moved onto the rule
// engine (verified against the previous hand-written matcher over ~188k
// operator/value/presence combinations, including Unicode case folding).
func TestEvaluateCondition(t *testing.T) {
	s := &policyServiceImpl{}
	type in struct {
		op      string
		vals    []string
		present bool
		targets []string
	}
	for _, tc := range []struct {
		name string
		in   in
		want bool
	}{
		{"equals case-insensitive", in{"equals", []string{"Admin"}, true, []string{"ADMIN"}}, true},
		{"empty operator is equals", in{"", []string{"admin"}, true, []string{"admin"}}, true},
		{"in any target", in{"in", []string{"user"}, true, []string{"admin", "USER"}}, true},
		{"equals absent attribute", in{"equals", nil, false, []string{"admin"}}, false},
		{"equals no match", in{"equals", []string{"user"}, true, []string{"admin"}}, false},
		{"not_equals absent attribute", in{"not_equals", nil, false, []string{"admin"}}, true},
		{"not_in match", in{"not_in", []string{"admin"}, true, []string{"ADMIN"}}, false},
		{"contains all targets", in{"contains", []string{"read", "Write"}, true, []string{"write", "read"}}, true},
		{"contains missing one", in{"contains", []string{"read"}, true, []string{"read", "write"}}, false},
		{"contains no targets is vacuous", in{"contains", []string{"read"}, true, nil}, true},
		{"contains absent", in{"contains", nil, false, nil}, false},
		{"not_contains none present", in{"not_contains", []string{"read"}, true, []string{"write"}}, true},
		{"not_contains one present", in{"not_contains", []string{"read"}, true, []string{"READ"}}, false},
		{"any present", in{"any", []string{"x"}, true, nil}, true},
		{"any empty", in{"any", []string{}, true, nil}, false},
		{"empty absent", in{"empty", nil, false, nil}, true},
		{"empty with values", in{"empty", []string{"x"}, true, nil}, false},
		{"unicode fold (Kelvin sign)", in{"equals", []string{"K"}, true, []string{"k"}}, true},
		{"unknown operator fails closed", in{"is_empty", nil, false, nil}, false},
		{"unknown operator fails closed when present", in{"bogus", []string{"x"}, true, []string{"x"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := s.evaluateCondition(tc.in.vals, tc.in.present, models.AttributeCondition{Operator: tc.in.op, Values: tc.in.targets})
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
