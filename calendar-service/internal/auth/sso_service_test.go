package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

func TestSAMLAutoProvisionDryRun(t *testing.T) {
	// Only run this test if requested, since it hits a real DB
	if os.Getenv("DRY_RUN_SSO") == "" {
		t.Skip("Skipping auto-provision dry run. Set DRY_RUN_SSO=1 to run.")
	}

	logger := logrus.New().WithField("test", "sso_dry_run")

	// Create Hasura client pointing to local dev instance
	hasuraURL := os.Getenv("HASURA_ENDPOINT")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080/v1/graphql"
	}
	hClient := hasura.NewClient(hasuraURL, os.Getenv("HASURA_ADMIN_SECRET"))

	// Since we need a tenant ID that exists, let's just query one first
	ctx := context.Background()
	var tenantResp struct {
		Tenants []struct {
			ID string `json:"id"`
		} `json:"tenants"`
	}
	err := hClient.QueryRaw(ctx, "query { tenants(limit: 1) { id } }", nil, &tenantResp)
	if err != nil {
		t.Fatalf("Failed to fetch tenant for test: %v", err)
	}
	if len(tenantResp.Tenants) == 0 {
		t.Fatalf("No tenants found in local DB. Please ensure a tenant exists.")
	}

	tenantID := tenantResp.Tenants[0].ID
	logger.Infof("Using Tenant ID: %s", tenantID)

	ssoService := NewSSOService(SSOServiceConfig{
		HasuraClient: hClient,
		Logger:       logger,
	})

	// Dry run: Provision a user payload
	email := "saml-dry-run-user@example.com"
	attributes := map[string]interface{}{
		"name":   "Jane Doe SAML",
		"groups": []string{"Engineering", "Enterprise-Users"},
	}

	logger.Info("Starting dry-run of SAML auto-provisioning payload...")

	payloadBytes, _ := json.MarshalIndent(attributes, "", "  ")
	logger.Infof("IdP Attributes Payload:\n%s", string(payloadBytes))

	user, err := ssoService.createUser(ctx, tenantID, email, attributes, "member")
	if err != nil {
		t.Fatalf("Dry run failed to create user: %v", err)
	}

	logger.Infof("Successfully auto-provisioned user! ID: %s, Email: %s", user.ID, user.Email)

	// Now try fetching the user back
	// Wait, getUserByIDPUser uses sso_sessions which requires a session. We'll skip that part
	// for the dry run and just assert the user exists in `users` table via query.
	var userResp struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	err = hClient.QueryRaw(ctx, fmt.Sprintf("query { users(where: {id: {_eq: \"%s\"}}) { id } }", user.ID), nil, &userResp)
	if err != nil {
		t.Fatalf("Verification query failed: %v", err)
	}
	if len(userResp.Users) == 0 {
		t.Fatalf("User was not found in the database after provision.")
	}
	logger.Infof("Verified user %s exists in db!", user.ID)
}
