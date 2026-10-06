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
// during archive mode=disable_consumers, and starts grain materialize for
// publish_version rewire_latest / bump_and_refresh. When unset, cascade falls
// back to direct SQL updates / skips materialize (unit tests / minimal wiring).
type CubeCascadeSideEffects struct {
	PauseSchedule      func(ctx context.Context, tenantID, scheduleID, actorUserID string) error
	DeactivatePipeline func(ctx context.Context, tenantID, pipelineID string) error
	StartMaterialize   func(ctx context.Context, req CubeMaterializeRequest) (workflowID string, plan *CubeMaterializePlan, err error)
}

// CubeCascadeRequest is the POST /api/cubes/{id}/cascade body.
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
	Action string `json:"action"` // disabled | deactivated | left_pin | archived | rewired | published | materialize_started | …
	Detail string `json:"detail,omitempty"`
}

// CubeCascadeMaterializeStart is one grain start attempt from publish cascade.
type CubeCascadeMaterializeStart struct {
	Grain          []string `json:"grain"`
	GrainHash      string   `json:"grainHash,omitempty"`
	WorkflowID     string   `json:"workflowId,omitempty"`
	AttemptID      string   `json:"attemptId,omitempty"`
	AlreadyRunning bool     `json:"alreadyRunning,omitempty"`
	Error          string   `json:"error,omitempty"`
	Skipped        bool     `json:"skipped,omitempty"`
}

// CubeCascadeReceipt is the success body for POST …/cascade.
type CubeCascadeReceipt struct {
	Cube              CubeDefinition                 `json:"cube"`
	ChangeClass       string                         `json:"changeClass"`
	Mode              string                         `json:"mode"`
	Action            string                         `json:"action"`
	ConsumersAffected []CubeCascadeAffected          `json:"consumersAffected"`
	Impact            *CubeImpactPreviewReport       `json:"impact,omitempty"`
	BreakReasons      []string                       `json:"breakReasons,omitempty"`
	PreviousVersion   int                            `json:"previousVersion,omitempty"`
	MaterializeStarts []CubeCascadeMaterializeStart  `json:"materializeStarts,omitempty"`
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
	errCascadeInvalidMode = errors.New("mode not allowed for this action/changeClass")
	errCascadeTokenReuse  = errors.New("confirm token already used")
	errCascadePatchOnly   = errors.New("action=patch is preview-only; cascade apply supports archive and publish_version")
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
	// Merge non-nil hooks so schedule wiring and materialize wiring can each
	// contribute without clobbering the other.
	if fx.PauseSchedule != nil {
		h.cascadeFX.PauseSchedule = fx.PauseSchedule
	}
	if fx.DeactivatePipeline != nil {
		h.cascadeFX.DeactivatePipeline = fx.DeactivatePipeline
	}
	if fx.StartMaterialize != nil {
		h.cascadeFX.StartMaterialize = fx.StartMaterialize
	}
}

// ApplyCubeCascade dispatches archive (A4) or publish_version (A5) cascade.
func (h *CubeHandler) ApplyCubeCascade(ctx context.Context, tenantID, userID, cubeID string, req CubeCascadeRequest) (*CubeCascadeReceipt, error) {
	action := strings.TrimSpace(strings.ToLower(req.Action))
	switch action {
	case CubeImpactActionArchive:
		return h.ApplyCubeCascadeArchive(ctx, tenantID, userID, cubeID, req)
	case CubeImpactActionPublishVersion:
		return h.ApplyCubeCascadePublishVersion(ctx, tenantID, userID, cubeID, req)
	case CubeImpactActionPatch:
		return nil, errCascadePatchOnly
	default:
		return nil, fmt.Errorf("action must be archive or publish_version")
	}
}

// ApplyCubeCascadeArchive runs A4 archive cascade for the given mode.
func (h *CubeHandler) ApplyCubeCascadeArchive(ctx context.Context, tenantID, userID, cubeID string, req CubeCascadeRequest) (*CubeCascadeReceipt, error) {
	action := strings.TrimSpace(strings.ToLower(req.Action))
	mode := strings.TrimSpace(strings.ToLower(req.Mode))
	if mode == "" {
		mode = CubeImpactModeFailClosed
	}
	if action != CubeImpactActionArchive {
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

// ApplyCubeCascadePublishVersion runs A5 breaking publish cascade.
func (h *CubeHandler) ApplyCubeCascadePublishVersion(ctx context.Context, tenantID, userID, cubeID string, req CubeCascadeRequest) (*CubeCascadeReceipt, error) {
	mode := strings.TrimSpace(strings.ToLower(req.Mode))
	if mode == "" {
		mode = CubeImpactModeFailClosed
	}

	preview, err := h.BuildCubeImpactPreview(ctx, tenantID, cubeID, CubeImpactPreviewRequest{
		Action: CubeImpactActionPublishVersion,
		Patch:  req.Patch,
	})
	if err != nil {
		return nil, err
	}
	if preview.ChangeClass != CubeImpactClassBreakingContract {
		return nil, fmt.Errorf("draft is not a breaking change; use PATCH /api/cubes/{id}")
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
		CubeImpactActionPublishVersion,
		preview.Composition.ContentHash,
		preview.PatchHash,
	)
	if err != nil {
		return nil, fmt.Errorf("confirm token: %w", err)
	}

	if mode == CubeImpactModeFailClosed && preview.BlockingCount > 0 {
		return nil, &ErrCubeCascadeBlocked{Impact: preview}
	}

	cube, err := h.getByID(ctx, tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	if cube.TenantID != tenantID {
		return nil, errors.New("cannot version a core cube owned by the gold-copy tenant")
	}
	if cube.IsCore && !h.isGoldCopy(ctx, tenantID) {
		return nil, errors.New("cannot version a core cube")
	}
	if cube.Status == "archived" || cube.ArchivedAt != nil {
		return nil, errors.New("cube is archived")
	}

	draft := *cube
	if req.Patch != nil {
		draft = req.Patch.applyPatch(*cube)
	}
	reasons := DetectCubeContractBreaking(*cube, draft)
	if len(reasons) == 0 {
		return nil, fmt.Errorf("draft is not a breaking change; use PATCH /api/cubes/{id}")
	}
	if err := h.validateForWrite(ctx, tenantID, &draft); err != nil {
		return nil, err
	}
	prevVersion := cube.ContractVersion
	nextVersion := prevVersion + 1
	draft.ContractVersion = nextVersion

	if err := markConfirmTokenUsed(strings.TrimSpace(req.ConfirmToken), claims.Exp); err != nil {
		return nil, err
	}

	out, err := h.updateCube(ctx, draft)
	if err != nil {
		return nil, fmt.Errorf("publish cube version: %w", err)
	}

	affected := []CubeCascadeAffected{{
		Kind:   "cube",
		ID:     out.ID,
		Label:  out.Name,
		Action: "published",
		Detail: fmt.Sprintf("contract_version %d → %d", prevVersion, nextVersion),
	}}

	var starts []CubeCascadeMaterializeStart
	switch mode {
	case CubeImpactModeFailClosed:
		// publish only; pins keep old version until operator rewires
	case CubeImpactModeRewireLatest:
		rewired, rerr := h.rewirePublishConsumers(ctx, tenantID, cubeID, nextVersion, preview.Consumers)
		if rerr != nil {
			return nil, rerr
		}
		affected = append(affected, rewired...)
		starts = h.startCascadeMaterialize(ctx, tenantID, cubeID, out.Grains)
		for _, s := range starts {
			action := "materialize_started"
			detail := s.WorkflowID
			if s.Skipped {
				action = "materialize_skipped"
				detail = "no materialize starter configured"
			} else if s.Error != "" {
				action = "materialize_error"
				detail = s.Error
			} else if s.AlreadyRunning {
				action = "materialize_already_running"
			}
			affected = append(affected, CubeCascadeAffected{
				Kind:   "grain",
				ID:     strings.Join(s.Grain, ","),
				Action: action,
				Detail: detail,
			})
		}
	case CubeImpactModeBumpAndRefresh:
		for _, c := range preview.Consumers {
			if c.Kind == "saved_query" || c.Kind == "page_tile" || c.Kind == "report" {
				affected = append(affected, CubeCascadeAffected{
					Kind:   c.Kind,
					ID:     c.ID,
					Label:  c.Label,
					Action: "left_pin",
					Detail: fmt.Sprintf("pins retained on prior contractVersion; new published=%d", nextVersion),
				})
			}
		}
		starts = h.startCascadeMaterialize(ctx, tenantID, cubeID, out.Grains)
		for _, s := range starts {
			action := "materialize_started"
			detail := s.WorkflowID
			if s.Skipped {
				action = "materialize_skipped"
				detail = "no materialize starter configured"
			} else if s.Error != "" {
				action = "materialize_error"
				detail = s.Error
			} else if s.AlreadyRunning {
				action = "materialize_already_running"
			}
			affected = append(affected, CubeCascadeAffected{
				Kind:   "grain",
				ID:     strings.Join(s.Grain, ","),
				Action: action,
				Detail: detail,
			})
		}
	default:
		return nil, errCascadeInvalidMode
	}

	reasonStrs := make([]string, 0, len(reasons))
	for _, r := range reasons {
		reasonStrs = append(reasonStrs, string(r))
	}

	return &CubeCascadeReceipt{
		Cube:              *out,
		ChangeClass:       CubeImpactClassBreakingContract,
		Mode:              mode,
		Action:            CubeImpactActionPublishVersion,
		ConsumersAffected: affected,
		Impact:            preview,
		BreakReasons:      reasonStrs,
		PreviousVersion:   prevVersion,
		MaterializeStarts: starts,
	}, nil
}

func (h *CubeHandler) startCascadeMaterialize(ctx context.Context, tenantID, cubeID string, grains [][]string) []CubeCascadeMaterializeStart {
	out := make([]CubeCascadeMaterializeStart, 0, len(grains))
	if len(grains) == 0 {
		return out
	}
	if h.cascadeFX.StartMaterialize == nil {
		for _, g := range grains {
			out = append(out, CubeCascadeMaterializeStart{Grain: g, Skipped: true})
		}
		return out
	}
	for _, g := range grains {
		wfID, plan, err := h.cascadeFX.StartMaterialize(ctx, CubeMaterializeRequest{
			TenantID: tenantID,
			CubeID:   cubeID,
			Grain:    g,
			Force:    true,
		})
		res := CubeCascadeMaterializeStart{Grain: g, WorkflowID: wfID}
		if plan != nil {
			res.GrainHash = plan.GrainHash
			res.AttemptID = plan.AttemptID
		}
		if err != nil {
			msg := err.Error()
			if strings.Contains(strings.ToLower(msg), "already") {
				res.AlreadyRunning = true
			}
			res.Error = msg
		}
		out = append(out, res)
	}
	return out
}

func (h *CubeHandler) rewirePublishConsumers(
	ctx context.Context,
	tenantID, cubeID string,
	nextVersion int,
	consumers []CubeImpactConsumer,
) ([]CubeCascadeAffected, error) {
	out := make([]CubeCascadeAffected, 0)
	for _, c := range consumers {
		switch c.Kind {
		case "saved_query":
			changed, err := h.rewireSavedQueryPin(ctx, tenantID, c.ID, cubeID, nextVersion)
			if err != nil {
				return out, fmt.Errorf("rewire saved_query %s: %w", c.ID, err)
			}
			action := "rewired"
			detail := fmt.Sprintf("subject.contractVersion → %d", nextVersion)
			if !changed {
				action = "rewire_noop"
				detail = "subject already at target version or missing cube subject"
			}
			out = append(out, CubeCascadeAffected{
				Kind: c.Kind, ID: c.ID, Label: c.Label, Action: action, Detail: detail,
			})
		case "page_tile":
			changed, err := h.rewirePageCubePins(ctx, tenantID, c.ID, cubeID, nextVersion)
			if err != nil {
				return out, fmt.Errorf("rewire page %s: %w", c.ID, err)
			}
			action := "rewired"
			detail := fmt.Sprintf("mirrored subject.contractVersion → %d", nextVersion)
			if !changed {
				action = "rewire_noop"
				detail = "no matching cube subject pins found in page JSON"
			}
			out = append(out, CubeCascadeAffected{
				Kind: c.Kind, ID: c.ID, Label: c.Label, Action: action, Detail: detail,
			})
		case "report":
			changed, err := h.rewireReportCubePin(ctx, tenantID, c.ID, cubeID, nextVersion)
			if err != nil {
				return out, fmt.Errorf("rewire report %s: %w", c.ID, err)
			}
			action := "rewired"
			detail := fmt.Sprintf("subject.contractVersion → %d", nextVersion)
			if !changed {
				action = "rewire_noop"
				detail = "no matching cube subject pin (legacy name-only left unchanged)"
			}
			out = append(out, CubeCascadeAffected{
				Kind: c.Kind, ID: c.ID, Label: c.Label, Action: action, Detail: detail,
			})
		}
	}
	return out, nil
}

func (h *CubeHandler) rewireSavedQueryPin(ctx context.Context, tenantID, sqID, cubeID string, nextVersion int) (bool, error) {
	var raw []byte
	err := h.db.GetContext(ctx, &raw, `
		SELECT query_state FROM data_explorer.saved_query
		WHERE id::text = $1 AND tenant_id = $2 AND archived_at IS NULL
	`, sqID, tenantID)
	if err != nil {
		return false, err
	}
	updated, changed, err := rewireCubeSubjectsInJSON(raw, cubeID, nextVersion)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	_, err = h.db.ExecContext(ctx, `
		UPDATE data_explorer.saved_query
		SET query_state = $3::jsonb, updated_at = now()
		WHERE id::text = $1 AND tenant_id = $2
	`, sqID, tenantID, string(updated))
	return err == nil && changed, err
}

func (h *CubeHandler) rewirePageCubePins(ctx context.Context, tenantID, pageID, cubeID string, nextVersion int) (bool, error) {
	var components, appModel []byte
	err := h.db.QueryRowContext(ctx, `
		SELECT COALESCE(components, 'null'::jsonb), COALESCE(app_model, 'null'::jsonb)
		FROM page_definitions
		WHERE id::text = $1 AND tenant_id = $2::uuid
	`, pageID, tenantID).Scan(&components, &appModel)
	if err != nil {
		return false, err
	}
	c2, cChanged, err := rewireCubeSubjectsInJSON(components, cubeID, nextVersion)
	if err != nil {
		return false, err
	}
	a2, aChanged, err := rewireCubeSubjectsInJSON(appModel, cubeID, nextVersion)
	if err != nil {
		return false, err
	}
	if !cChanged && !aChanged {
		return false, nil
	}
	_, err = h.db.ExecContext(ctx, `
		UPDATE page_definitions
		SET components = $3::jsonb, app_model = $4::jsonb, updated_at = now()
		WHERE id::text = $1 AND tenant_id = $2::uuid
	`, pageID, tenantID, string(c2), string(a2))
	return err == nil && (cChanged || aChanged), err
}

func (h *CubeHandler) rewireReportCubePin(ctx context.Context, tenantID, reportID, cubeID string, nextVersion int) (bool, error) {
	var raw []byte
	err := h.db.GetContext(ctx, &raw, `
		SELECT definition FROM report_definitions
		WHERE id::text = $1 AND tenant_id = $2::uuid
	`, reportID, tenantID)
	if err != nil {
		return false, err
	}
	updated, changed, err := rewireCubeSubjectsInJSON(raw, cubeID, nextVersion)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	_, err = h.db.ExecContext(ctx, `
		UPDATE report_definitions
		SET definition = $3::jsonb, updated_at = now()
		WHERE id::text = $1 AND tenant_id = $2::uuid
	`, reportID, tenantID, string(updated))
	return err == nil && changed, err
}

// rewireCubeSubjectsInJSON walks JSON and sets contractVersion on any object
// with kind=cube and matching cubeId. Never remaps cubeId to another cube.
func rewireCubeSubjectsInJSON(raw []byte, cubeID string, nextVersion int) ([]byte, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, false, nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw, false, err
	}
	changed := rewireCubeSubjectsValue(v, cubeID, nextVersion)
	if !changed {
		return raw, false, nil
	}
	out, err := json.Marshal(v)
	return out, true, err
}

func rewireCubeSubjectsValue(v interface{}, cubeID string, nextVersion int) bool {
	changed := false
	switch t := v.(type) {
	case map[string]interface{}:
		kind, _ := t["kind"].(string)
		cid, _ := t["cubeId"].(string)
		if strings.EqualFold(kind, "cube") && cid == cubeID {
			cur := t["contractVersion"]
			need := true
			switch n := cur.(type) {
			case float64:
				need = int(n) != nextVersion
			case json.Number:
				i, err := n.Int64()
				need = err != nil || int(i) != nextVersion
			case string:
				need = true // including "latest" → numeric pin on publish policy
			case nil:
				need = true
			}
			if need {
				t["contractVersion"] = float64(nextVersion)
				changed = true
			}
		}
		for _, child := range t {
			if rewireCubeSubjectsValue(child, cubeID, nextVersion) {
				changed = true
			}
		}
	case []interface{}:
		for _, child := range t {
			if rewireCubeSubjectsValue(child, cubeID, nextVersion) {
				changed = true
			}
		}
	}
	return changed
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

	receipt, err := h.ApplyCubeCascade(r.Context(), secCtx.TenantID, secCtx.UserID, id, req)
	if err == sql.ErrNoRows {
		h.writeError(w, fmt.Errorf("cube not found"), http.StatusNotFound)
		return
	}
	var blocked *ErrCubeCascadeBlocked
	if errors.As(err, &blocked) {
		changeClass := CubeImpactClassArchive
		if blocked.Impact != nil && blocked.Impact.ChangeClass != "" {
			changeClass = blocked.Impact.ChangeClass
		}
		h.writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error":         "cascade blocked",
			"details":       blocked.Error(),
			"changeClass":   changeClass,
			"mode":          CubeImpactModeFailClosed,
			"blockingCount": blocked.Impact.BlockingCount,
			"impact":        blocked.Impact,
		})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errCascadePatchOnly),
			errors.Is(err, errCascadeInvalidMode),
			strings.Contains(msg, "action must be"),
			strings.Contains(msg, "request body"),
			strings.Contains(msg, "not a breaking change"):
			status = http.StatusBadRequest
		case errors.Is(err, errCascadeTokenReuse),
			strings.Contains(msg, "confirm token"),
			strings.Contains(msg, "already archived"),
			strings.Contains(msg, "cube is archived"):
			status = http.StatusConflict
		case strings.Contains(msg, "cannot modify a core cube"),
			strings.Contains(msg, "cannot version a core cube"),
			strings.Contains(msg, "gold-copy"):
			status = http.StatusForbidden
		}
		h.writeError(w, err, status)
		return
	}
	h.writeJSON(w, http.StatusOK, receipt)
}
