package services

import (
	"testing"

	"github.com/hondyman/semlayer/backend/internal/models"
)

func TestPolicyServiceAllowsByRole(t *testing.T) {
	svc := NewPolicyService()

	user := models.User{
		ID:          "user-1",
		Role:        "Steward",
		Roles:       []string{"Steward"},
		Permissions: []string{"bundle:publish"},
		Attributes: map[string]string{
			"region": "EMEA",
		},
	}

	policies := []models.Policy{{
		ID:          "allow-read",
		Effect:      "allow",
		Actions:     []string{"read"},
		Resources:   []string{"bundle:nav-risk"},
		Description: "Stewards can read assigned bundles",
		Conditions: []models.AttributeCondition{{
			Attribute: "roles",
			Operator:  "contains",
			Values:    []string{"Steward"},
		}},
	}}

	allowed, err := svc.Can(user, "read", "bundle:nav-risk", policies)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatalf("expected access to be allowed")
	}
}

func TestPolicyServiceMatchesAttributeAndPermission(t *testing.T) {
	svc := NewPolicyService()

	user := models.User{
		ID:          "user-2",
		Role:        "Analyst",
		Roles:       []string{"Analyst"},
		Permissions: []string{"bundle:publish"},
		Attributes: map[string]string{
			"region": "APAC",
		},
	}

	policies := []models.Policy{
		{
			ID:          "allow-publish",
			Effect:      "allow",
			Actions:     []string{"publish"},
			Resources:   []string{"bundle:client-a"},
			Description: "Users with publish permission may publish",
			Conditions: []models.AttributeCondition{{
				Attribute: "permissions",
				Operator:  "contains",
				Values:    []string{"bundle:publish"},
			}},
		},
		{
			ID:          "region-match",
			Effect:      "allow",
			Actions:     []string{"publish"},
			Resources:   []string{"bundle:client-a"},
			Description: "Region-specific publish rights",
			Conditions: []models.AttributeCondition{{
				Attribute: "attributes.region",
				Operator:  "equals",
				Values:    []string{"APAC"},
			}},
		},
	}

	allowed, err := svc.Can(user, "publish", "bundle:client-a", policies)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatalf("expected publish to be allowed")
	}
}

func TestPolicyServiceDeniesOnDenyPolicy(t *testing.T) {
	svc := NewPolicyService()

	user := models.User{
		ID:    "user-3",
		Role:  "Viewer",
		Roles: []string{"Viewer"},
	}

	policies := []models.Policy{
		{
			ID:          "deny-read",
			Effect:      "deny",
			Actions:     []string{"read"},
			Resources:   []string{"bundle:sensitive"},
			Description: "Viewers may not read sensitive bundles",
			Conditions: []models.AttributeCondition{{
				Attribute: "roles",
				Operator:  "contains",
				Values:    []string{"Viewer"},
			}},
		},
	}

	allowed, err := svc.Can(user, "read", "bundle:sensitive", policies)
	if err == nil {
		t.Fatalf("expected error for deny policy, got nil")
	}
	if allowed {
		t.Fatalf("expected access to be denied")
	}
}
