package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

type dbModulesEnvelope struct {
	Measures   []models.SemanticObjectReference `json:"measures,omitempty"`
	Dimensions []models.SemanticObjectReference `json:"dimensions,omitempty"`
	Filters    []string                         `json:"filters,omitempty"`
}

type dbPoliciesEnvelope struct {
	Row    []models.BundleRowPolicy    `json:"row,omitempty"`
	Column []models.BundleColumnPolicy `json:"column,omitempty"`
}

type dbGovernanceEnvelope struct {
	Status       string              `json:"status,omitempty"`
	StewardGroup string              `json:"steward_group,omitempty"`
	Metadata     map[string]any      `json:"metadata,omitempty"`
	Policies     *dbPoliciesEnvelope `json:"policies,omitempty"`
	Extra        map[string]any      `json:"-"`
}

func bundleFromDBRow(bundleID, name, audience, version string, modules, metrics, governance sql.NullString, isActive bool, createdAt, updatedAt time.Time) (*models.DataBundle, error) {
	bundle := &models.DataBundle{
		ID:          bundleID,
		Name:        name,
		Description: name,
		Owner:       "",
		Version:     version,
		Status:      models.StatusDraft,
		Audience:    normalizeAudience(audience),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Manifest: models.BundleManifest{
			Identifier: bundleID,
			Summary:    name,
		},
		Lifecycle: models.BundleLifecycle{
			DraftedAt: createdAt,
		},
		Composition: models.BundleComposition{},
	}

	if modules.Valid && strings.TrimSpace(modules.String) != "" {
		var modEnv dbModulesEnvelope
		if err := json.Unmarshal([]byte(modules.String), &modEnv); err == nil {
			if len(modEnv.Measures) > 0 {
				bundle.Measures = append([]models.SemanticObjectReference{}, modEnv.Measures...)
				bundle.Composition.Measures = append([]models.SemanticObjectReference{}, modEnv.Measures...)
			}
			if len(modEnv.Dimensions) > 0 {
				bundle.Dimensions = append([]models.SemanticObjectReference{}, modEnv.Dimensions...)
				bundle.Composition.Dimensions = append([]models.SemanticObjectReference{}, modEnv.Dimensions...)
			}
			if len(modEnv.Filters) > 0 {
				bundle.Composition.Filters = append([]string{}, modEnv.Filters...)
			}
		}
	}

	if governance.Valid && strings.TrimSpace(governance.String) != "" {
		var govEnv dbGovernanceEnvelope
		if err := json.Unmarshal([]byte(governance.String), &govEnv); err == nil {
			if govEnv.Metadata != nil {
				if desc, ok := govEnv.Metadata["description"].(string); ok && strings.TrimSpace(desc) != "" {
					bundle.Description = strings.TrimSpace(desc)
					bundle.Manifest.Summary = bundle.Description
				}
				if owner, ok := govEnv.Metadata["owner"].(string); ok {
					bundle.Owner = strings.TrimSpace(owner)
				}
				if allowed, ok := govEnv.Metadata["allowedRoles"]; ok {
					bundle.AllowedRoles = uniqueStrings(stringSliceFromAny(allowed))
				}
				if aud, ok := govEnv.Metadata["audience"]; ok {
					if slice := stringSliceFromAny(aud); len(slice) > 0 {
						bundle.Audience = uniqueStrings(slice)
					}
				}
			}
			if govEnv.Policies != nil {
				if len(govEnv.Policies.Row) > 0 {
					bundle.RowPolicies = sanitizeRowPolicies(govEnv.Policies.Row)
				}
				if len(govEnv.Policies.Column) > 0 {
					bundle.ColumnPolicies = sanitizeColumnPolicies(govEnv.Policies.Column)
				}
			}
			bundle.Status = normalizeBundleStatus(govEnv.Status, isActive)
		} else {
			bundle.Status = normalizeBundleStatus("", isActive)
		}
	} else {
		bundle.Status = normalizeBundleStatus("", isActive)
	}

	if bundle.Description == "" {
		bundle.Description = name
		bundle.Manifest.Summary = name
	}

	switch bundle.Status {
	case models.StatusDraft:
		bundle.Lifecycle.LastAction = "import"
		bundle.Lifecycle.LastNotes = "Loaded from database (draft)"
	case models.StatusCertified:
		bundle.Lifecycle.CertifiedAt = cloneTime(updatedAt)
		bundle.Lifecycle.LastAction = "certify"
	case models.StatusPublished:
		bundle.Lifecycle.PublishedAt = cloneTime(updatedAt)
		bundle.PublishedAt = cloneTime(updatedAt)
		bundle.Lifecycle.LastAction = "publish"
	case models.StatusDeprecated:
		bundle.Lifecycle.DeprecatedAt = cloneTime(updatedAt)
		bundle.DeprecatedAt = cloneTime(updatedAt)
		bundle.Lifecycle.LastAction = "deprecate"
	}

	return bundle, nil
}

func serializeBundleForDB(bundle *models.DataBundle) ([]byte, []byte, []byte, string, bool, time.Time, time.Time, error) {
	if bundle == nil {
		return nil, nil, nil, "", true, time.Time{}, time.Time{}, fmt.Errorf("bundle is nil")
	}

	measures := bundle.Measures
	if len(measures) == 0 && len(bundle.Composition.Measures) > 0 {
		measures = bundle.Composition.Measures
	}
	dimensions := bundle.Dimensions
	if len(dimensions) == 0 && len(bundle.Composition.Dimensions) > 0 {
		dimensions = bundle.Composition.Dimensions
	}
	filters := bundle.Composition.Filters

	modEnv := dbModulesEnvelope{
		Measures:   copySemanticRefs(measures),
		Dimensions: copySemanticRefs(dimensions),
		Filters:    append([]string{}, filters...),
	}
	modulesJSON, err := json.Marshal(modEnv)
	if err != nil {
		return nil, nil, nil, "", true, time.Time{}, time.Time{}, fmt.Errorf("marshal modules: %w", err)
	}

	metricsJSON, err := json.Marshal([]any{})
	if err != nil {
		return nil, nil, nil, "", true, time.Time{}, time.Time{}, fmt.Errorf("marshal metrics: %w", err)
	}

	sanitizedRows := sanitizeRowPolicies(bundle.RowPolicies)
	sanitizedColumns := sanitizeColumnPolicies(bundle.ColumnPolicies)
	var policies *dbPoliciesEnvelope
	if len(sanitizedRows) > 0 || len(sanitizedColumns) > 0 {
		policies = &dbPoliciesEnvelope{Row: sanitizedRows, Column: sanitizedColumns}
	}

	metadata := map[string]any{}
	if strings.TrimSpace(bundle.Description) != "" {
		metadata["description"] = strings.TrimSpace(bundle.Description)
	}
	if strings.TrimSpace(bundle.Owner) != "" {
		metadata["owner"] = strings.TrimSpace(bundle.Owner)
	}
	if len(bundle.Audience) > 0 {
		metadata["audience"] = uniqueStrings(bundle.Audience)
	}
	if len(bundle.AllowedRoles) > 0 {
		metadata["allowedRoles"] = uniqueStrings(bundle.AllowedRoles)
	}

	govEnv := dbGovernanceEnvelope{
		Status:   strings.ToLower(string(bundle.Status)),
		Metadata: metadata,
		Policies: policies,
	}
	governanceJSON, err := json.Marshal(govEnv)
	if err != nil {
		return nil, nil, nil, "", true, time.Time{}, time.Time{}, fmt.Errorf("marshal governance: %w", err)
	}

	audience := "lp"
	if len(bundle.Audience) > 0 && strings.TrimSpace(bundle.Audience[0]) != "" {
		audience = strings.TrimSpace(bundle.Audience[0])
	}

	isActive := bundle.Status != models.StatusDeprecated

	createdAt := bundle.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := bundle.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return modulesJSON, metricsJSON, governanceJSON, audience, isActive, createdAt.UTC(), updatedAt.UTC(), nil
}

func normalizeBundleStatus(raw string, isActive bool) models.BundleStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "draft":
		return models.StatusDraft
	case "certified":
		return models.StatusCertified
	case "published", "active":
		return models.StatusPublished
	case "deprecated", "inactive", "retired":
		return models.StatusDeprecated
	default:
		if !isActive {
			return models.StatusDeprecated
		}
		return models.StatusPublished
	}
}

func normalizeAudience(audience string) []string {
	audience = strings.TrimSpace(audience)
	if audience == "" {
		return []string{"lp"}
	}
	return []string{audience}
}

func stringSliceFromAny(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string{}, v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func copySemanticRefs(values []models.SemanticObjectReference) []models.SemanticObjectReference {
	if len(values) == 0 {
		return nil
	}
	out := make([]models.SemanticObjectReference, len(values))
	copy(out, values)
	return out
}

func cloneTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	tt := t
	return &tt
}
