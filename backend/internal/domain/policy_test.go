package domain

import (
	"context"
	"testing"
)

// _fakePolicyRepo test scaffolding removed — unused in current tests.

type strictChecker struct {
	isCertified func(assetID string) bool
}

func (p *strictChecker) Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]any, []string, error) {
	has := false
	for _, c := range claims {
		if c.AssetID == req.AssetID && (c.Permission == req.Action || req.Action == PermRead && c.Permission == PermRead) {
			has = true
			break
		}
	}
	if !has {
		return false, "No effective claim", []map[string]any{{"policyId": "missing_claim", "result": "fail"}}, nil, nil
	}
	if p.isCertified(req.AssetID) && (req.Action == PermUpdate || req.Action == PermDelete) {
		return false, "Update/Delete on certified asset requires approval", []map[string]any{{"policyId": "cert_update_block", "result": "fail"}}, nil, nil
	}
	return true, "Allowed", []map[string]any{{"policyId": "baseline", "result": "pass"}}, []string{"all"}, nil
}

func TestPolicyChecker_TableDriven(t *testing.T) {
	checker := &strictChecker{isCertified: func(assetID string) bool {
		return assetID == "asset-certified-metric"
	}}

	cases := []struct {
		name      string
		req       EvaluationRequest
		claims    []EffectiveClaim
		wantAllow bool
	}{
		{"allow_read_certified", EvaluationRequest{UserID: "u", TenantID: "t", AssetID: "asset-certified-metric", Action: PermRead},
			[]EffectiveClaim{{AssetID: "asset-certified-metric", Permission: PermRead, Scope: nil, Source: "role"}}, true},
		{"deny_update_certified", EvaluationRequest{UserID: "u", TenantID: "t", AssetID: "asset-certified-metric", Action: PermUpdate},
			[]EffectiveClaim{{AssetID: "asset-certified-metric", Permission: PermUpdate, Scope: nil, Source: "manual"}}, false},
		{"allow_update_uncertified", EvaluationRequest{UserID: "u", TenantID: "t", AssetID: "asset-raw-view", Action: PermUpdate},
			[]EffectiveClaim{{AssetID: "asset-raw-view", Permission: PermUpdate, Scope: nil, Source: "manual"}}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok, _, _, _, err := checker.Check(context.Background(), c.req, c.claims)
			if err != nil {
				t.Fatal(err)
			}
			if ok != c.wantAllow {
				t.Fatalf("allow=%v want=%v", ok, c.wantAllow)
			}
		})
	}
}
