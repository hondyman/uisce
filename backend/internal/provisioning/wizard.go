package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// The wizard creates one tenant with its first instance. Its input is validated
// here, before any side effect. Activity inputs carry references only. The LDAP
// bind password goes to the secret store, and the workflow receives only the path.

const (
	wizardMaxBody        = 64 << 10
	wizardMaxAllowlist   = 256
	wizardMaxNameLen     = 200
	wizardMaxPasswordLen = 256
	identityPathLeaf     = "identity"
	ldapPasswordKey      = "ldap_bind_password"
)

var (
	wizardHostname = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	wizardDN       = regexp.MustCompile(`^[A-Za-z0-9=,. _-]{1,512}$`)
	wizardEnvs     = map[string]bool{"dev": true, "uat": true, "prod": true}
)

// RegionChecker reports whether a region code is configured. It must fail closed.
type RegionChecker interface {
	Known(code string) bool
}

// IdentitySecrets writes secret values. It is the subset of the secret store the
// wizard needs.
type IdentitySecrets interface {
	PutMap(ctx context.Context, key string, values map[string]string) error
}

// WizardStarter starts the onboarding workflow. The real implementation, which
// starts the parent workflow, lands with #429. Tests use a fake.
type WizardStarter interface {
	StartWizard(ctx context.Context, in WizardInput) (workflowID string, err error)
}

// WizardInput is what the workflow receives. It holds references only: no
// password, no URL.
type WizardInput struct {
	TenantID    string
	InstanceID  string
	TenantCode  string
	TenantName  string
	Environment string
	Region      string
	Allowlist   []string
	LDAP        *WizardLDAP
}

// WizardLDAP is the non-secret LDAP settings. The bind password is in the store.
type WizardLDAP struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	BaseDN string `json:"base_dn"`
	BindDN string `json:"bind_dn"`
}

type wizardRequest struct {
	TenantCode  string      `json:"tenant_code"`
	TenantName  string      `json:"tenant_name"`
	Environment string      `json:"environment"`
	Region      string      `json:"region"`
	Allowlist   []string    `json:"allowlist"`
	LDAP        *wizardLDAP `json:"ldap"`
}

type wizardLDAP struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	BaseDN       string `json:"base_dn"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
}

// WizardResponse is returned with 202. It never contains a secret.
type WizardResponse struct {
	WorkflowID string `json:"workflow_id"`
	TenantID   string `json:"tenant_id"`
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`
}

// WizardHandler serves the tenant-creation wizard endpoint.
type WizardHandler struct {
	DB      *sql.DB
	Starter WizardStarter
	Regions RegionChecker
	Secrets IdentitySecrets
}

// RegisterWizardRoutes mounts the endpoint. It is admin-only through admin().
func (h *WizardHandler) RegisterWizardRoutes(r chi.Router) {
	r.Post("/system/tenants/wizard", h.Create)
}

// Create validates the request, writes the LDAP bind password to the store if
// present, and starts the workflow. Nothing is written before validation passes.
func (h *WizardHandler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := admin(w, r); !ok {
		return
	}
	if h.DB == nil || h.Starter == nil || h.Regions == nil || h.Secrets == nil {
		http.Error(w, "tenant wizard is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, wizardMaxBody))
	if err != nil {
		http.Error(w, "request body is too large or unreadable", http.StatusBadRequest)
		return
	}
	var req wizardRequest
	dec := json.NewDecoder(bytesReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateWizard(req, h.Regions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exists, err := h.tenantExists(r.Context(), req.TenantCode, req.TenantName)
	if err != nil {
		http.Error(w, "could not check for an existing tenant", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "a tenant with this code or name already exists", http.StatusConflict)
		return
	}

	in := WizardInput{
		TenantID:    uuid.NewString(),
		InstanceID:  uuid.NewString(),
		TenantCode:  req.TenantCode,
		TenantName:  req.TenantName,
		Environment: req.Environment,
		Region:      req.Region,
		Allowlist:   req.Allowlist,
	}
	if req.LDAP != nil {
		in.LDAP = &WizardLDAP{Host: req.LDAP.Host, Port: req.LDAP.Port, BaseDN: req.LDAP.BaseDN, BindDN: req.LDAP.BindDN}
		// The password goes to the store before the workflow starts. The workflow
		// reads it from there, so it never appears in the workflow input.
		path := "tenants/" + req.TenantCode + "/" + identityPathLeaf
		if err := h.Secrets.PutMap(r.Context(), path, map[string]string{ldapPasswordKey: req.LDAP.BindPassword}); err != nil {
			http.Error(w, "could not store the LDAP credential", http.StatusInternalServerError)
			return
		}
	}

	workflowID, err := h.Starter.StartWizard(r.Context(), in)
	if err != nil {
		http.Error(w, "could not start tenant provisioning", http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusAccepted, WizardResponse{
		WorkflowID: workflowID, TenantID: in.TenantID, InstanceID: in.InstanceID, Status: "provisioning",
	})
}

// tenantExists checks the code and the name. It fails closed: a database error is
// an error, never "not found", unlike the older handler helpers.
func (h *WizardHandler) tenantExists(ctx context.Context, code, name string) (bool, error) {
	var n int
	if err := h.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM public.tenants WHERE code = $1 OR LOWER(name) = LOWER($2)`, code, name).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func validateWizard(req wizardRequest, regions RegionChecker) error {
	if !codePattern.MatchString(req.TenantCode) {
		return errors.New("tenant_code must start with a letter and use only lowercase letters, digits and underscores")
	}
	if req.TenantName == "" || len(req.TenantName) > wizardMaxNameLen {
		return fmt.Errorf("tenant_name is required and at most %d characters", wizardMaxNameLen)
	}
	if !wizardEnvs[req.Environment] {
		return errors.New("environment must be dev, uat or prod")
	}
	if !regions.Known(req.Region) {
		return errors.New("region is not configured")
	}
	if len(req.Allowlist) > wizardMaxAllowlist {
		return fmt.Errorf("at most %d allowlist entries", wizardMaxAllowlist)
	}
	for _, e := range req.Allowlist {
		if !validAllowEntry(e) {
			// Report the rule, never the entry: entries are client input.
			return errors.New("allowlist entries must be IP addresses or CIDRs")
		}
	}
	if req.LDAP != nil {
		if err := validateWizardLDAP(*req.LDAP); err != nil {
			return err
		}
	}
	return nil
}

func validAllowEntry(e string) bool {
	if e == "" || len(e) > 64 {
		return false
	}
	if _, _, err := net.ParseCIDR(e); err == nil {
		return true
	}
	return net.ParseIP(e) != nil
}

func validateWizardLDAP(l wizardLDAP) error {
	if !wizardHostname.MatchString(l.Host) && net.ParseIP(l.Host) == nil {
		return errors.New("ldap host must be a hostname or IP address")
	}
	if l.Port < 1 || l.Port > 65535 {
		return errors.New("ldap port must be between 1 and 65535")
	}
	if !wizardDN.MatchString(l.BaseDN) || !wizardDN.MatchString(l.BindDN) {
		return errors.New("ldap base_dn and bind_dn must be plain distinguished names")
	}
	if l.BindPassword == "" || len(l.BindPassword) > wizardMaxPasswordLen {
		return fmt.Errorf("ldap bind_password must be 1 to %d characters", wizardMaxPasswordLen)
	}
	return nil
}
