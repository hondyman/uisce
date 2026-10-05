package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

// CubeCascadeSideEffects optionally pauses schedules / deactivates pipelines
// during archive mode=disable_consumers. When unset, cascade falls back to
// direct SQL updates (unit tests / minimal wiring).
type CubeCascadeSideEffects struct {
	PauseSchedule      func(ctx context.Context, tenantID, scheduleID, actorUserID string) error
	DeactivatePipeline func(ctx context.Context, tenantID, pipelineID string) error
}

// CubeCascadeRequest is the POST /api/cubes/{id}/cascade body (A4 archive;
// A5 adds publish_version modes).
type CubeCascadeRequest struct {
	Action       string            `json:"action"`
	ConfirmToken string            `json:"confirmToken"`
	Mode         string            `json:"mode"`
	Patch        *cubeWriteRequest `json:"patch,omitempty"`
}

// CubeCascadeAffected is one consumer touched by cascade apply.
type CubeCascadeAffected struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Label  string `json:"label,omitempty"`
	Action string `json:"action"` // disabled | deactivated | left_pin | archived
	Detail string `json:"detail,omitempty"`
}

// CubeCascadeReceipt is the success body for POST …/cascade.
type CubeCascadeReceipt struct {
	Cube              CubeDefinition         `json:"cube"`
	ChangeClass       string                 `json:"changeClass"`
	Mode              string                 `json:"mode"`
	Action            string                 `json:"action"`
	ConsumersAffected []CubeCascadeAffected  `json:"consumersAffected"`
	Impact            *CubeImpactPreviewReport `json:"impact,omitempty"`
}

// ErrCubeCascadeBlocked is returned when fail_closed finds blocking consumers.
type ErrCubeCascadeBlocked struct {
	Impact *CubeImpactPreviewReport
}

func (e *ErrCubeCascadeBlocked) Error() string {
	n := 0
	if e.Impact != nil {
		n = e.Impact.BlockingCount
	}
	return fmt.Sprintf("cascade blocked: %d blocking consumer(s); use a documented cascade mode or resolve consumers", n)
}

var (
	errCascadeInvalidMode   = errors.New("mode not allowed for this action/changeClass")
	errCascadeTokenReuse    = errors.New("confirm token already used")
	errCascadeActionA5      = errors.New("publish_version cascade lands in A5; A4 supports action=archive only")
	errCascadePatchNotA4    = errors.New("action=patch is preview-only; cascade apply supports archive (A4) and publish_version (A5)")
)

// usedConfirmTokens tracks single-use confirm tokens for this process.
// Keyed by full token string; values are expiry unix for opportunistic prune.
var usedConfirmTokens sync.Map

func markConfirmTokenUsed(token string, exp int64) error {
	if token == "" {
		return errors.New("confirm token required")
	}
	if _, loaded := usedConfirmTokens.LoadOrStore(token, exp); loaded {
		return errCascadeTokenReuse
	}
	// Opportunistic prune of expired entries.
	now := time.Now().UTC().Unix()
	usedConfirmTokens.Range(func(k, v interface{}) bool {
		if exp, ok := v.(int64); ok && exp > 0 && exp < now {
			usedConfirmTokens.Delete(k)
		}
		return true
	})
	return nil
}

func (h *CubeHandler) SetCascadeSideEffects(fx CubeCascadeSideEffects) {
	h.cascadeFX = fx
}

// ApplyCubeCascadeArchive runs A4 archive cascade for the given mode.
func (h *CubeHandler) ApplyCubeCascadeArchive(ctx context.Context, tenantID, userID, cubeID string, req CubeCascadeRequest) (*CubeCascadeReceipt, error) {
	action := strings.TrimSpace(strings.ToLower(req.Action))
	mode := strings.TrimSpace(strings.ToLower(req.Mode))
	if mode == "" {
		mode = CubeImpactModeFailClosed
	}

	switch action {
	case CubeImpactActionArchive:
	case CubeImpactActionPublishVersion:
		return nil, errCascadeActionA5
	case CubeImpactActionPatch:
		return nil, errCascadePatchNotA4
	default:
		return nil, fmt.Errorf("action must be archive")
	}

	preview, err := h.BuildCubeImpactPreview(ctx, tenantID, cubeID, CubeImpactPreviewRequest{
		Action: CubeImpactActionArchive,
		Patch:  req.Patch,
	})
	if err != nil {
		return nil, err
	}

	allowed := false
	for _, m := range preview.AllowedModes {
		if m == mode {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errCascadeInvalidMode
	}

	key, err := h.impactConfirmKey()
	if err != nil {
		return nil, err
	}
	claims, err := VerifyCubeImpactConfirmToken(
		key,
		strings.TrimSpace(req.ConfirmToken),
		tenantID,
		cubeID,
		CubeImpactActionArchive,
		preview.Composition.ContentHash,
		preview.PatchHash,
	)
	if err != nil {
		return nil, fmt.Errorf("confirm token: %w", err)
	}

	if mode == CubeImpactModeFailClosed && preview.BlockingCount > 0 {
		// Do not consume the token on a blocked dry-run so the operator can
		// retry with disable_consumers using the same preview token.
		return nil, &ErrCubeCascadeBlocked{Impact: preview}
	}

	cube, err := h.getByID(ctx, tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	if cube.TenantID != tenantID {
		return nil, errors.New("cannot modify a core cube owned by the gold-copy tenant")
	}
	if cube.IsCore && !h.isGoldCopy(ctx, tenantID) {
		return nil, errors.New("cannot modify a core cube")
	}
	if cube.Status == "archived" || cube.ArchivedAt != nil {
		return nil, errors.New("cube already archived")
	}

	if err := markConfirmTokenUsed(strings.TrimSpace(req.ConfirmToken), claims.Exp); err != nil {
		return nil, err
	}

	affected := make([]CubeCascadeAffected, 0)
	switch mode {
	case CubeImpactModeFailClosed:
		// no side-effects; archive below
	case CubeImpactModeDisableConsumers:
		disabled, derr := h.disableArchiveConsumers(ctx, tenantID, userID, cubeID, preview.Consumers)
		if derr != nil {
			return nil, derr
		}
		affected = append(affected, disabled...)
		// SQ / page pins intentionally left; runtime fail-closed on archived cubes.
		for _, c := range preview.Consumers {
			if c.Kind == "saved_query" || c.Kind == "page_tile" || c.Kind == "report" {
				affected = append(affected, CubeCascadeAffected{
					Kind:   c.Kind,
					ID:     c.ID,
					Label:  c.Label,
					Action: "left_pin",
					Detail: "pins retained; runtime fail-closed on archived cube",
				})
			}
		}
	default:
		return nil, errCascadeInvalidMode
	}

	draft := *cube
	draft.Status = "archived"
	out, err := h.updateCube(ctx, draft)
	if err != nil {
		return nil, fmt.Errorf("archive cube: %w", err)
	}
	affected = append(affected, CubeCascadeAffected{
		Kind:   "cube",
		ID:     out.ID,
		Label:  out.Name,
		Action: "archived",
	})

	return &CubeCascadeReceipt{
		Cube:              *out,
		ChangeClass:       CubeImpactClassArchive,
		Mode:              mode,
		Action:            CubeImpactActionArchive,
		ConsumersAffected: affected,
		Impact:            preview,
	}, nil
}

func (h *CubeHandler) disableArchiveConsumers(
	ctx context.Context,
	tenantID, userID, cubeID string,
	consumers []CubeImpactConsumer,
) ([]CubeCascadeAffected, error) {
	out := make([]CubeCascadeAffected, 0)
	for _, c := range consumers {
		switch c.Kind {
		case "schedule":
			if c.Enabled != nil && !*c.Enabled {
				continue
			}
			if err := h.pauseCubeSchedule(ctx, tenantID, c.ID, userID); err != nil {
				return out, fmt.Errorf("pause schedule %s: %w", c.ID, err)
			}
			out = append(out, CubeCascadeAffected{
				Kind:   "schedule",
				ID:     c.ID,
				Label:  c.Label,
				Action: "disabled",
				Detail: "cube_refresh paused for archive cascade",
			})
		case "pipeline":
			if c.Enabled != nil && !*c.Enabled {
				continue
			}
			if err := h.deactivateCubePipeline(ctx, tenantID, c.ID); err != nil {
				return out, fmt.Errorf("deactivate pipeline %s: %w", c.ID, err)
			}
			out = append(out, CubeCascadeAffected{
				Kind:   "pipeline",
				ID:     c.ID,
				Label:  c.Label,
				Action: "deactivated",
				Detail: "cube_materialize pipeline set inactive for archive cascade",
			})
		}
	}
	_ = cubeID
	return out, nil
}

func (h *CubeHandler) pauseCubeSchedule(ctx context.Context, tenantID, scheduleID, actorUserID string) error {
	if h.cascadeFX.PauseSchedule != nil {
		return h.cascadeFX.PauseSchedule(ctx, tenantID, scheduleID, actorUserID)
	}
	_, err := h.db.ExecContext(ctx, `
		UPDATE public.schedules
		SET enabled = false, updated_by = $3, updated_at = now(), version = version + 1
		WHERE id::text = $1 AND tenant_id::text = $2 AND deleted_at IS NULL
	`, scheduleID, tenantID, nullActor(actorUserID))
	return err
}

func (h *CubeHandler) deactivateCubePipeline(ctx context.Context, tenantID, pipelineID string) error {
	if h.cascadeFX.DeactivatePipeline != nil {
		return h.cascadeFX.DeactivatePipeline(ctx, tenantID, pipelineID)
	}
	_, err := h.db.ExecContext(ctx, `
		UPDATE data_pipeline_definitions
		SET is_active = false, last_modified_at = now()
		WHERE id::text = $1 AND tenant_id = $2::uuid
	`, pipelineID, tenantID)
	return err
}

func nullActor(userID string) string {
	if strings.TrimSpace(userID) == "" {
		return "system:cube-cascade"
	}
	return userID
}

// HandlePostCubeCascade handles POST /api/cubes/{id}/cascade.
func (h *CubeHandler) HandlePostCubeCascade(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
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
	var req CubeCascadeRequest
	if len(strings.TrimSpace(string(raw))) == 0 {
		h.writeError(w, fmt.Errorf("request body required"), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	receipt, err := h.ApplyCubeCascadeArchive(r.Context(), secCtx.TenantID, secCtx.UserID, id, req)
	if err == sql.ErrNoRows {
		h.writeError(w, fmt.Errorf("cube not found"), http.StatusNotFound)
		return
	}
	var blocked *ErrCubeCascadeBlocked
	if errors.As(err, &blocked) {
		h.writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error":          "cascade blocked",
			"details":        blocked.Error(),
			"changeClass":    CubeImpactClassArchive,
			"mode":           CubeImpactModeFailClosed,
			"blockingCount":  blocked.Impact.BlockingCount,
			"impact":         blocked.Impact,
		})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errCascadeActionA5),
			errors.Is(err, errCascadePatchNotA4),
			errors.Is(err, errCascadeInvalidMode),
			strings.Contains(msg, "action must be"),
			strings.Contains(msg, "request body"):
			status = http.StatusBadRequest
		case errors.Is(err, errCascadeTokenReuse),
			strings.Contains(msg, "confirm token"),
			strings.Contains(msg, "already archived"):
			status = http.StatusConflict
		case strings.Contains(msg, "cannot modify a core cube"),
			strings.Contains(msg, "gold-copy"):
			status = http.StatusForbidden
		}
		h.writeError(w, err, status)
		return
	}
	h.writeJSON(w, http.StatusOK, receipt)
}
