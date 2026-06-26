package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/values"
	"github.com/jmoiron/sqlx"
)

type ValuesAuditLogger interface {
	LogDataModification(ctx context.Context, actorID, tenantID, objectID, objectType, objectName, action string, oldData, newData interface{}) error
}

type ValuesService interface {
	CreateValueTheme(ctx context.Context, name, description string) (*values.ValueTheme, error)
	GetValueThemes(ctx context.Context) ([]*values.ValueTheme, error)

	CreateValueSignal(ctx context.Context, input ValueSignalInput) (*values.ValueSignal, error)
	GetValueSignals(ctx context.Context, issuerID string) ([]*values.ValueSignal, error)

	CreateClientValuesProfile(ctx context.Context, clientID string, templateID *uuid.UUID) (*values.ClientValuesProfile, error)
	GetClientValuesProfile(ctx context.Context, clientID string) (*values.ClientValuesProfile, error)
	UpdateClientValuesProfile(ctx context.Context, clientID string, preferences json.RawMessage) (*values.ClientValuesProfile, error)

	CreateConstraint(ctx context.Context, input ConstraintInput) (*values.Constraint, error)
	GetConstraints(ctx context.Context, profileID uuid.UUID) ([]*values.Constraint, error)
}

type ValueSignalInput struct {
	IssuerID     string                   `json:"issuer_id"`
	InstrumentID *string                  `json:"instrument_id"`
	ThemeID      uuid.UUID                `json:"theme_id"`
	SourceID     uuid.UUID                `json:"source_id"`
	Score        float64                  `json:"score"`
	Summary      string                   `json:"summary"`
	EvidenceRefs []values.EvidenceRef     `json:"evidence_refs"`
	Status       values.ValueSignalStatus `json:"status"`
	Confidence   float64                  `json:"confidence"`
	ValidUntil   *time.Time               `json:"valid_until"`
}

type ConstraintInput struct {
	ClientValuesProfileID *uuid.UUID                `json:"client_values_profile_id"`
	StrategyTemplateID    *uuid.UUID                `json:"strategy_template_id"`
	Name                  string                    `json:"name"`
	Scope                 values.ConstraintScope    `json:"scope"`
	Operator              values.ConstraintOperator `json:"operator"`
	Condition             json.RawMessage           `json:"condition"`
	Severity              values.ConstraintSeverity `json:"severity"`
	Priority              int                       `json:"priority"`
}

type valuesServiceImpl struct {
	db       *sqlx.DB
	auditSvc ValuesAuditLogger
}

func NewValuesService(db *sqlx.DB, auditSvc ValuesAuditLogger) ValuesService {
	return &valuesServiceImpl{db: db, auditSvc: auditSvc}
}

func (s *valuesServiceImpl) CreateValueTheme(ctx context.Context, name, description string) (*values.ValueTheme, error) {
	theme := &values.ValueTheme{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO value_themes (id, name, description, created_at, updated_at)
		VALUES (:id, :name, :description, :created_at, :updated_at)
	`
	_, err := s.db.NamedExecContext(ctx, query, theme)
	if err != nil {
		return nil, fmt.Errorf("failed to create value theme: %w", err)
	}

	return theme, nil
}

func (s *valuesServiceImpl) GetValueThemes(ctx context.Context) ([]*values.ValueTheme, error) {
	var themes []*values.ValueTheme
	query := `SELECT * FROM value_themes ORDER BY name`
	err := s.db.SelectContext(ctx, &themes, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get value themes: %w", err)
	}
	return themes, nil
}

func (s *valuesServiceImpl) CreateValueSignal(ctx context.Context, input ValueSignalInput) (*values.ValueSignal, error) {
	evidenceJSON, err := json.Marshal(input.EvidenceRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evidence refs: %w", err)
	}

	signal := &values.ValueSignal{
		ID:           uuid.New(),
		IssuerID:     input.IssuerID,
		InstrumentID: input.InstrumentID,
		ThemeID:      input.ThemeID,
		SourceID:     input.SourceID,
		Score:        input.Score,
		Summary:      input.Summary,
		EvidenceRefs: input.EvidenceRefs, // Note: Struct field, but DB needs JSONB
		Status:       input.Status,
		Confidence:   input.Confidence,
		ValidFrom:    time.Now(),
		ValidUntil:   input.ValidUntil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// We need a struct that matches the DB columns exactly for NamedExec, specifically for JSONB fields if they are not automatically handled by sqlx/driver
	// The pq driver usually handles []byte or string for JSONB.
	// Let's define a temporary struct or map for insertion if needed.
	// Actually, if the struct field has `db` tag, sqlx tries to map it.
	// But `EvidenceRefs` is `[]EvidenceRef`. The driver might not know how to convert slice to JSONB automatically unless it implements Valuer.
	// A safer way is to pass a map or a struct with []byte/string for JSON fields.

	type dbSignal struct {
		*values.ValueSignal
		EvidenceRefsJSON []byte `db:"evidence_refs"`
	}

	dbSig := dbSignal{
		ValueSignal:      signal,
		EvidenceRefsJSON: evidenceJSON,
	}

	query := `
		INSERT INTO value_signals (
			id, issuer_id, instrument_id, theme_id, source_id, score, summary, evidence_refs, status, confidence, valid_from, valid_until, created_at, updated_at
		) VALUES (
			:id, :issuer_id, :instrument_id, :theme_id, :source_id, :score, :summary, :evidence_refs, :status, :confidence, :valid_from, :valid_until, :created_at, :updated_at
		)
	`
	_, err = s.db.NamedExecContext(ctx, query, dbSig)
	if err != nil {
		return nil, fmt.Errorf("failed to create value signal: %w", err)
	}

	return signal, nil
}

func (s *valuesServiceImpl) GetValueSignals(ctx context.Context, issuerID string) ([]*values.ValueSignal, error) {
	var signals []*values.ValueSignal
	// We need to handle JSONB scanning.
	// If the struct field is []EvidenceRef, we need the driver/sqlx to handle it.
	// Usually we implement Scanner interface on a custom type, or scan into a temporary struct.
	// For simplicity, let's scan into a struct with []byte for JSONB fields and then unmarshal.

	type dbSignal struct {
		values.ValueSignal
		EvidenceRefsJSON []byte `db:"evidence_refs"`
	}

	var dbSignals []dbSignal
	query := `SELECT * FROM value_signals WHERE issuer_id = $1`
	err := s.db.SelectContext(ctx, &dbSignals, query, issuerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get value signals: %w", err)
	}

	for _, dbs := range dbSignals {
		sig := dbs.ValueSignal
		if len(dbs.EvidenceRefsJSON) > 0 {
			if err := json.Unmarshal(dbs.EvidenceRefsJSON, &sig.EvidenceRefs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal evidence refs: %w", err)
			}
		}
		signals = append(signals, &sig)
	}

	return signals, nil
}

func (s *valuesServiceImpl) CreateClientValuesProfile(ctx context.Context, clientID string, templateID *uuid.UUID) (*values.ClientValuesProfile, error) {
	profile := &values.ClientValuesProfile{
		ID:                 uuid.New(),
		ClientID:           clientID,
		StrategyTemplateID: templateID,
		Preferences:        json.RawMessage("{}"),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	query := `
		INSERT INTO client_values_profiles (id, client_id, strategy_template_id, preferences, created_at, updated_at)
		VALUES (:id, :client_id, :strategy_template_id, :preferences, :created_at, :updated_at)
	`
	_, err := s.db.NamedExecContext(ctx, query, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create client values profile: %w", err)
	}

	// Audit Log
	if s.auditSvc != nil {
		s.auditSvc.LogDataModification(ctx, "system", clientID, profile.ID.String(), "ClientValuesProfile", "Profile", "CREATE", nil, profile)
	}

	return profile, nil
}

func (s *valuesServiceImpl) GetClientValuesProfile(ctx context.Context, clientID string) (*values.ClientValuesProfile, error) {
	var profile values.ClientValuesProfile
	query := `SELECT * FROM client_values_profiles WHERE client_id = $1`
	err := s.db.GetContext(ctx, &profile, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client values profile: %w", err)
	}
	return &profile, nil
}

func (s *valuesServiceImpl) UpdateClientValuesProfile(ctx context.Context, clientID string, preferences json.RawMessage) (*values.ClientValuesProfile, error) {
	query := `
		UPDATE client_values_profiles
		SET preferences = :preferences, updated_at = :updated_at
		WHERE client_id = :client_id
		RETURNING *
	`

	// NamedExec doesn't support RETURNING easily with sqlx structs in one go usually,
	// but we can use NamedQuery or just Exec and then Get.
	// Let's use Exec and Get.

	args := map[string]interface{}{
		"client_id":   clientID,
		"preferences": preferences,
		"updated_at":  time.Now(),
	}

	_, err := s.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to update client values profile: %w", err)
	}

	updatedProfile, err := s.GetClientValuesProfile(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Audit Log
	if s.auditSvc != nil {
		s.auditSvc.LogDataModification(ctx, "system", clientID, updatedProfile.ID.String(), "ClientValuesProfile", "Profile", "UPDATE", nil, updatedProfile)
	}

	return updatedProfile, nil
}

func (s *valuesServiceImpl) CreateConstraint(ctx context.Context, input ConstraintInput) (*values.Constraint, error) {
	scopeJSON, err := json.Marshal(input.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scope: %w", err)
	}

	constraint := &values.Constraint{
		ID:                    uuid.New(),
		ClientValuesProfileID: input.ClientValuesProfileID,
		StrategyTemplateID:    input.StrategyTemplateID,
		Name:                  input.Name,
		Scope:                 input.Scope,
		Operator:              input.Operator,
		Condition:             input.Condition,
		Severity:              input.Severity,
		Priority:              input.Priority,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	type dbConstraint struct {
		*values.Constraint
		ScopeJSON []byte `db:"scope"`
	}

	dbC := dbConstraint{
		Constraint: constraint,
		ScopeJSON:  scopeJSON,
	}

	query := `
		INSERT INTO constraints (
			id, client_values_profile_id, strategy_template_id, name, scope, operator, condition, severity, priority, created_at, updated_at
		) VALUES (
			:id, :client_values_profile_id, :strategy_template_id, :name, :scope, :operator, :condition, :severity, :priority, :created_at, :updated_at
		)
	`
	_, err = s.db.NamedExecContext(ctx, query, dbC)
	if err != nil {
		return nil, fmt.Errorf("failed to create constraint: %w", err)
	}

	// Audit Log
	if s.auditSvc != nil {
		// Determine tenant/client ID from profile if possible, here assuming system/unknown for MVP or fetching profile
		s.auditSvc.LogDataModification(ctx, "system", "unknown", constraint.ID.String(), "Constraint", constraint.Name, "CREATE", nil, constraint)
	}

	return constraint, nil
}

func (s *valuesServiceImpl) GetConstraints(ctx context.Context, profileID uuid.UUID) ([]*values.Constraint, error) {
	var constraints []*values.Constraint

	type dbConstraint struct {
		values.Constraint
		ScopeJSON []byte `db:"scope"`
	}

	var dbConstraints []dbConstraint
	query := `SELECT * FROM constraints WHERE client_values_profile_id = $1 ORDER BY priority DESC`
	err := s.db.SelectContext(ctx, &dbConstraints, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}

	for _, dbc := range dbConstraints {
		c := dbc.Constraint
		if len(dbc.ScopeJSON) > 0 {
			if err := json.Unmarshal(dbc.ScopeJSON, &c.Scope); err != nil {
				return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
			}
		}
		constraints = append(constraints, &c)
	}

	return constraints, nil
}
