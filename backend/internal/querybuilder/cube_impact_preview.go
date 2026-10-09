package querybuilder

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

const (
	cubeImpactConfirmTTL = 10 * time.Minute

	CubeImpactActionArchive        = "archive"
	CubeImpactActionPatch          = "patch"
	CubeImpactActionPublishVersion = "publish_version"

	CubeImpactClassArchive           = "archive"
	CubeImpactClassNonBreakingPatch  = "non_breaking_patch"
	CubeImpactClassBreakingContract  = "breaking_contract"

	CubeImpactModeFailClosed       = "fail_closed"
	CubeImpactModeDisableConsumers = "disable_consumers"
	CubeImpactModeRewireLatest     = "rewire_latest"
	CubeImpactModeBumpAndRefresh   = "bump_and_refresh"
)

// CubeImpactPreviewRequest is the POST /api/cubes/{id}/impact/preview body.
type CubeImpactPreviewRequest struct {
	Action string            `json:"action"`
	Patch  *cubeWriteRequest `json:"patch,omitempty"`
}

// CubeImpactPreviewReport extends the inventory with change classification.
type CubeImpactPreviewReport struct {
	CubeImpactReport
	ChangeClass     string   `json:"changeClass"`
	BreakReasons    []string `json:"breakReasons"`
	BlockingCount   int      `json:"blockingCount"`
	ConfirmToken    string   `json:"confirmToken"`
	AllowedModes    []string `json:"allowedModes"`
	RecommendedMode string   `json:"recommendedMode"`
	NextVersion     int      `json:"nextVersion,omitempty"`
	PatchHash       string   `json:"patchHash,omitempty"`
}

type cubeImpactConfirmClaims struct {
	CubeID      string `json:"cubeId"`
	TenantID    string `json:"tenantId"`
	Action      string `json:"action"`
	ContentHash string `json:"contentHash"`
	PatchHash   string `json:"patchHash,omitempty"`
	ChangeClass string `json:"changeClass"`
	Exp         int64  `json:"exp"`
}

func (h *CubeHandler) impactConfirmKey() ([]byte, error) {
	if len(h.impactConfirmSecret) > 0 {
		return h.impactConfirmSecret, nil
	}
	if s := strings.TrimSpace(os.Getenv("CUBE_IMPACT_CONFIRM_SECRET")); s != "" {
		return []byte(s), nil
	}
	if s := strings.TrimSpace(os.Getenv("JWT_SECRET")); s != "" {
		return []byte(s), nil
	}
	return nil, errors.New("CUBE_IMPACT_CONFIRM_SECRET or JWT_SECRET required to mint impact confirm tokens")
}

func mintCubeImpactConfirmToken(key []byte, claims cubeImpactConfirmClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyCubeImpactConfirmToken validates an A2 confirm token for cascade (A4+).
func VerifyCubeImpactConfirmToken(key []byte, token, tenantID, cubeID, action, contentHash, patchHash string) (*cubeImpactConfirmClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid confirm token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid confirm token payload")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid confirm token signature")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, errors.New("confirm token signature mismatch")
	}
	var claims cubeImpactConfirmClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("invalid confirm token claims")
	}
	if claims.Exp > 0 && time.Now().UTC().Unix() > claims.Exp {
		return nil, errors.New("confirm token expired")
	}
	if claims.TenantID != tenantID || claims.CubeID != cubeID {
		return nil, errors.New("confirm token cube/tenant mismatch")
	}
	if claims.Action != action {
		return nil, errors.New("confirm token action mismatch")
	}
	if claims.ContentHash != contentHash {
		return nil, errors.New("confirm token contentHash mismatch")
	}
	if patchHash != "" || claims.PatchHash != "" {
		if claims.PatchHash != patchHash {
			return nil, errors.New("confirm token patchHash mismatch")
		}
	}
	return &claims, nil
}

func hashCubeImpactPatch(action string, draft *CubeDefinition, patch *cubeWriteRequest) string {
	type surface struct {
		Action          string                     `json:"action"`
		Name            string                     `json:"name,omitempty"`
		BOID            string                     `json:"boId,omitempty"`
		Dimensions      []CubeDimension            `json:"dimensions,omitempty"`
		TimeDimension   *CubeTimeDimension         `json:"timeDimension,omitempty"`
		MetricIDs       []string                   `json:"metricIds,omitempty"`
		Grains          [][]string                 `json:"grains,omitempty"`
		Materialization *CubeMaterializationConfig `json:"materialization,omitempty"`
		Federation      *CubeFederation            `json:"federation,omitempty"`
		Status          string                     `json:"status,omitempty"`
		DraftVersion    int                        `json:"draftContractVersion,omitempty"`
	}
	s := surface{Action: action}
	if draft != nil {
		s.Name = draft.Name
		s.BOID = draft.BOID
		s.Dimensions = draft.Dimensions
		s.TimeDimension = draft.TimeDimension
		s.MetricIDs = draft.MetricIDs
		s.Grains = draft.Grains
		mat := draft.Materialization
		s.Materialization = &mat
		fed := draft.Federation
		s.Federation = &fed
		s.Status = draft.Status
		s.DraftVersion = draft.ContractVersion
	}
	if action == CubeImpactActionArchive {
		s.Status = "archived"
	}
	_ = patch // draft already incorporates patch via applyPatch
	raw, _ := json.Marshal(s)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func allowedModesForChangeClass(changeClass string) []string {
	switch changeClass {
	case CubeImpactClassArchive:
		return []string{CubeImpactModeFailClosed, CubeImpactModeDisableConsumers}
	case CubeImpactClassBreakingContract:
		return []string{CubeImpactModeFailClosed, CubeImpactModeRewireLatest, CubeImpactModeBumpAndRefresh}
	default:
		return []string{CubeImpactModeFailClosed}
	}
}

func pinContractVersion(c CubeImpactConsumer) (num int, latest bool, present bool) {
	if c.Pin == nil || c.Pin.ContractVersion == nil {
		return 0, false, false
	}
	switch v := c.Pin.ContractVersion.(type) {
	case float64:
		return int(v), false, true
	case int:
		return v, false, true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false, true
		}
		return int(n), false, true
	case string:
		if strings.EqualFold(v, "latest") {
			return 0, true, true
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, true
		}
		return n, false, true
	default:
		return 0, false, true
	}
}

// reclassifyConsumersForPreview adjusts severity/blocking for the proposed action.
// A1 inventory uses archive-oriented defaults; preview rewrites for patch/publish.
func reclassifyConsumersForPreview(action, changeClass string, consumers []CubeImpactConsumer, nextVersion int) []CubeImpactConsumer {
	out := make([]CubeImpactConsumer, len(consumers))
	copy(out, consumers)
	for i := range out {
		c := &out[i]
		switch changeClass {
		case CubeImpactClassNonBreakingPatch:
			c.Severity = "info"
			c.Blocking = false
			if c.Detail == "" {
				c.Detail = "non-breaking patch; cascade not required"
			}
		case CubeImpactClassArchive:
			// Keep A1 archive-oriented classification.
		case CubeImpactClassBreakingContract:
			switch c.Kind {
			case "saved_query":
				n, isLatest, hasPin := pinContractVersion(*c)
				if isLatest {
					c.Severity = "warning"
					c.Blocking = false
					c.Detail = "pin=latest; will follow new contract after publish"
				} else if hasPin && n == nextVersion {
					c.Severity = "info"
					c.Blocking = false
					c.Detail = fmt.Sprintf("already pinned to next contractVersion=%d", nextVersion)
				} else {
					c.Severity = "blocking"
					c.Blocking = true
					if hasPin {
						c.Detail = fmt.Sprintf("pin contractVersion=%v ≠ next %d", c.Pin.ContractVersion, nextVersion)
					} else {
						c.Detail = fmt.Sprintf("cube saved query needs pin rewire to %d", nextVersion)
					}
				}
			case "page_tile":
				if strings.HasPrefix(c.Detail, "status=") && !strings.Contains(c.Detail, "published") {
					c.Severity = "info"
					c.Blocking = false
				} else if c.Severity == "warning" && !c.Blocking {
					c.Severity = "info"
				} else {
					c.Severity = "blocking"
					c.Blocking = true
					c.Detail = "published page tile may pin old contractVersion"
				}
			case "schedule", "pipeline":
				c.Severity = "warning"
				c.Blocking = false
				if c.Enabled != nil && !*c.Enabled {
					c.Severity = "info"
				} else {
					c.Detail = "will target new contract on next refresh/materialize"
				}
			case "report":
				if c.Blocking && strings.Contains(c.Detail, "subject") {
					c.Severity = "blocking"
					c.Blocking = true
					c.Detail = "report subject pin may need rewire"
				} else {
					c.Severity = "info"
					c.Blocking = false
				}
			default:
				c.Severity = "warning"
				c.Blocking = false
			}
		}
		_ = action
	}
	return out
}

func countBlocking(consumers []CubeImpactConsumer) (blocking, warning int) {
	for _, c := range consumers {
		if c.Blocking {
			blocking++
		}
		if c.Severity == "warning" {
			warning++
		}
	}
	return blocking, warning
}

// BuildCubeImpactPreview classifies a proposed archive/patch/publish against live consumers.
func (h *CubeHandler) BuildCubeImpactPreview(ctx context.Context, tenantID, cubeID string, req CubeImpactPreviewRequest) (*CubeImpactPreviewReport, error) {
	action := strings.TrimSpace(strings.ToLower(req.Action))
	switch action {
	case CubeImpactActionArchive, CubeImpactActionPatch, CubeImpactActionPublishVersion:
	default:
		return nil, fmt.Errorf("action must be archive, patch, or publish_version")
	}

	base, err := h.BuildCubeImpact(ctx, tenantID, cubeID, true)
	if err != nil {
		return nil, err
	}

	cube, err := h.getByID(ctx, tenantID, cubeID)
	if err != nil {
		return nil, err
	}

	draft := *cube
	if req.Patch != nil && (action == CubeImpactActionPatch || action == CubeImpactActionPublishVersion) {
		draft = req.Patch.applyPatch(*cube)
	}

	var reasons []CubeContractBreakReason
	changeClass := CubeImpactClassNonBreakingPatch
	nextVersion := cube.ContractVersion

	switch action {
	case CubeImpactActionArchive:
		changeClass = CubeImpactClassArchive
		draft.Status = "archived"
	case CubeImpactActionPatch, CubeImpactActionPublishVersion:
		reasons = DetectCubeContractBreaking(*cube, draft)
		if len(reasons) > 0 {
			changeClass = CubeImpactClassBreakingContract
			nextVersion = cube.ContractVersion + 1
		} else {
			changeClass = CubeImpactClassNonBreakingPatch
		}
		if action == CubeImpactActionPublishVersion && len(reasons) == 0 {
			return nil, fmt.Errorf("draft is not a breaking change; use action=patch")
		}
	}

	reasonStrs := make([]string, 0, len(reasons))
	for _, r := range reasons {
		reasonStrs = append(reasonStrs, string(r))
	}

	consumers := reclassifyConsumersForPreview(action, changeClass, base.Consumers, nextVersion)
	blocking, warning := countBlocking(consumers)
	base.Consumers = consumers
	base.Summary.BlockingCount = blocking
	base.Summary.WarningCount = warning
	base.Summary.ConsumerCount = len(consumers)

	patchHash := hashCubeImpactPatch(action, &draft, req.Patch)
	key, err := h.impactConfirmKey()
	if err != nil {
		return nil, err
	}
	token, err := mintCubeImpactConfirmToken(key, cubeImpactConfirmClaims{
		CubeID:      cube.ID,
		TenantID:    tenantID,
		Action:      action,
		ContentHash: cube.ContentHash,
		PatchHash:   patchHash,
		ChangeClass: changeClass,
		Exp:         time.Now().UTC().Add(cubeImpactConfirmTTL).Unix(),
	})
	if err != nil {
		return nil, err
	}

	out := &CubeImpactPreviewReport{
		CubeImpactReport: *base,
		ChangeClass:      changeClass,
		BreakReasons:     reasonStrs,
		BlockingCount:    blocking,
		ConfirmToken:     token,
		AllowedModes:     allowedModesForChangeClass(changeClass),
		RecommendedMode:  CubeImpactModeFailClosed,
		PatchHash:        patchHash,
	}
	if changeClass == CubeImpactClassBreakingContract {
		out.NextVersion = nextVersion
	}
	return out, nil
}

// HandlePostCubeImpactPreview handles POST /api/cubes/{id}/impact/preview.
func (h *CubeHandler) HandlePostCubeImpactPreview(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, handlers.SecurityErrorStatus(err))
		return
	}
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		h.writeError(w, fmt.Errorf("cube id is required"), http.StatusBadRequest)
		return
	}
	raw, err := readLimitedBody(r, 1<<20)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var req CubeImpactPreviewRequest
	if len(strings.TrimSpace(string(raw))) == 0 {
		h.writeError(w, fmt.Errorf("request body required"), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	report, err := h.BuildCubeImpactPreview(r.Context(), secCtx.TenantID, id, req)
	if err == sql.ErrNoRows {
		h.writeError(w, fmt.Errorf("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusInternalServerError
		if strings.Contains(msg, "action must be") ||
			strings.Contains(msg, "draft is not a breaking change") ||
			strings.Contains(msg, "request body") {
			status = http.StatusBadRequest
		}
		if strings.Contains(msg, "JWT_SECRET") || strings.Contains(msg, "CONFIRM_SECRET") {
			status = http.StatusInternalServerError
		}
		h.writeError(w, err, status)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}
