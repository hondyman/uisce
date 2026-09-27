package datapipeline

import "testing"

// TestClassifyAction covers the full boolean space of the override classifier.
// The two pairs that share output but differ in `priorWasOverride` (cases 3
// vs. 4, cases 6 vs. 7) are the regression tests for the false-positive
// trap the design surfaced: "model updated without override" must not be
// misread as "override expired."
func TestClassifyAction(t *testing.T) {
	cases := []struct {
		name             string
		modelValue       string
		effectiveValue   string
		priorValue       string
		hasPrior         bool
		priorWasOverride bool
		wantKind         actionKind
		wantModelValue   *string
	}{
		{
			name:             "no override, no prior",
			modelValue:       "A-",
			effectiveValue:   "A-",
			priorValue:       "",
			hasPrior:         false,
			priorWasOverride: false,
			wantKind:         actionAffirm,
			wantModelValue:   nil,
		},
		{
			name:             "no override, prior match (no transition)",
			modelValue:       "A-",
			effectiveValue:   "A-",
			priorValue:       "A-",
			hasPrior:         true,
			priorWasOverride: false,
			wantKind:         actionAffirm,
			wantModelValue:   nil,
		},
		{
			name:             "no override, model changed, prior not override (model update, not expiry)",
			modelValue:       "A-",
			effectiveValue:   "A-",
			priorValue:       "BBB+",
			hasPrior:         true,
			priorWasOverride: false,
			wantKind:         actionAffirm,
			wantModelValue:   nil,
		},
		{
			name:             "no override, model changed, prior was override (expired)",
			modelValue:       "A-",
			effectiveValue:   "A-",
			priorValue:       "BBB+",
			hasPrior:         true,
			priorWasOverride: true,
			wantKind:         actionOverrideExpired,
			wantModelValue:   nil,
		},
		{
			name:             "override applies, no prior",
			modelValue:       "A-",
			effectiveValue:   "BBB+",
			priorValue:       "",
			hasPrior:         false,
			priorWasOverride: false,
			wantKind:         actionOverrideApplied,
			wantModelValue:   nil,
		},
		{
			name:             "override applies, prior was override, model != prior",
			modelValue:       "A-",
			effectiveValue:   "BB+",
			priorValue:       "BBB+",
			hasPrior:         true,
			priorWasOverride: true,
			wantKind:         actionOverrideApplied,
			wantModelValue:   strPtr("A-"),
		},
		{
			name:             "override applies, prior not override, model != prior",
			modelValue:       "A-",
			effectiveValue:   "BB+",
			priorValue:       "BBB+",
			hasPrior:         true,
			priorWasOverride: false,
			wantKind:         actionOverrideApplied,
			wantModelValue:   strPtr("A-"),
		},
		{
			name:             "override applies, prior exists, model == prior",
			modelValue:       "A-",
			effectiveValue:   "BBB+",
			priorValue:       "A-",
			hasPrior:         true,
			priorWasOverride: false,
			wantKind:         actionOverrideApplied,
			wantModelValue:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotModel := classifyAction(
				tc.modelValue, tc.effectiveValue, tc.priorValue,
				tc.hasPrior, tc.priorWasOverride,
			)
			if gotKind != tc.wantKind {
				t.Errorf("kind: got %q, want %q", gotKind, tc.wantKind)
			}
			if !strPtrEq(gotModel, tc.wantModelValue) {
				t.Errorf("model_value: got %v, want %v",
					derefStr(gotModel), derefStr(tc.wantModelValue))
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func strPtrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func derefStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
