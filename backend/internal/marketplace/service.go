package marketplace

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/jmoiron/sqlx"
)

// Service is the marketplace domain layer.  It orchestrates publish, browse,
// and install operations with transaction support, idempotency, and audit.
type Service struct {
	db *sqlx.DB
}

// NewService returns a marketplace Service backed by db.
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// PublishRequest is the domain-level input for publishing a new listing.
type PublishRequest struct {
	ActorID        uuid.UUID
	TenantID      uuid.UUID
	TenantName    string
	IdempotencyKey string
	RequestHash   []byte // SHA-256 of canonicalized JSON body
	Title         string
	Kind          ListingKind
	Category      string
	Description   string
	PriceCents    int
	BillingPeriod BillingPeriod
	Artifact      Artifact
}

// PublishResult is returned on a successful publish.
type PublishResult struct {
	ListingID string `json:"listing_id"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Version   string `json:"artifact_version"`
}

// Publish creates a listing and its first artifact version inside a single transaction.
// It is idempotent: if the same idempotency-key + request-hash is seen within 24 h,
// it returns the previously-generated result without error.
func (s *Service) Publish(ctx context.Context, req PublishRequest) (*PublishResult, *IdempotencyRecord, error) {
	repo := NewRepository(s.db)

	// ── 1. Idempotency check ──────────────────────────────────────────────
	if req.IdempotencyKey != "" {
		rec, found, err := repo.CheckIdempotency(ctx, req.IdempotencyKey,
			req.TenantID.String(), "publish")
		if err != nil {
			return nil, nil, fmt.Errorf("idempotency check: %w", err)
		}
		if found {
			if string(rec.RequestHash) != string(req.RequestHash) {
				return nil, rec, fmt.Errorf("idempotency key reused with different payload")
			}
			var result PublishResult
			if err := json.Unmarshal(rec.ResponseBody, &result); err != nil {
				return nil, nil, fmt.Errorf("unmarshal cached response: %w", err)
			}
			return &result, rec, nil
		}
	}

	// ── 2. Validate artifact and listing metadata ─────────────────────────
	if verrs := req.Artifact.Validate(); len(verrs) > 0 {
		return nil, nil, fmt.Errorf("artifact validation failed: %v", verrs)
	}
	if lerrs := ValidateListing(req.Kind, StatusPublished, req.BillingPeriod,
		req.Title, req.Description, req.PriceCents); len(lerrs) > 0 {
		return nil, nil, fmt.Errorf("listing validation failed: %v", lerrs)
	}

	// ── 3. Serialize artifact to canonical JSON and compute SHA-256 ──────────
	payloadBytes, err := MarshalPayload(req.Artifact)
	if err != nil {
		return nil, nil, fmt.Errorf("serialize artifact: %w", err)
	}
	h := sha256.Sum256(payloadBytes)
	req.Artifact.CanonicalSHA256 = hex.EncodeToString(h[:])

	// ── 4. Transaction: listing + artifact + idempotency record ────────────
	var publishResult *PublishResult
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	listingID := uuid.New()
	input := CreateListingInput{
		ID:                listingID,
		Title:             req.Title,
		Kind:              string(req.Kind),
		Category:          req.Category,
		PublisherTenantID: req.TenantID,
		PublisherName:     req.TenantName,
		PublisherActorID:  req.ActorID,
		Description:       req.Description,
		PriceCents:       req.PriceCents,
		BillingPeriod:     string(req.BillingPeriod),
		Status:            string(StatusPublished),
	}

	if _, err := CreateListing(ctx, tx, input); err != nil {
		return nil, nil, fmt.Errorf("create listing: %w", err)
	}

	artifactID, err := CreateArtifact(ctx, tx, listingID, payloadBytes, req.Artifact)
	if err != nil {
		return nil, nil, fmt.Errorf("create artifact: %w", err)
	}
	_ = artifactID

	if req.IdempotencyKey != "" {
		result := PublishResult{
			ListingID: listingID.String(),
			Title:     req.Title,
			Kind:      string(req.Kind),
			Version:   req.Artifact.ArtifactVersion,
		}
		body, _ := json.Marshal(result)
		rec := IdempotencyRecord{
			Key:            req.IdempotencyKey,
			TenantID:       req.TenantID.String(),
			Operation:      "publish",
			RequestHash:    req.RequestHash,
			ResponseStatus: http.StatusCreated,
			ResponseBody:   body,
		}
		if err := SaveIdempotency(ctx, tx, rec); err != nil {
			return nil, nil, fmt.Errorf("save idempotency: %w", err)
		}
	}

	publishResult = &PublishResult{
		ListingID: listingID.String(),
		Title:     req.Title,
		Kind:      string(req.Kind),
		Version:   req.Artifact.ArtifactVersion,
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}

	// ── 5. Async audit (fire after commit) ─────────────────────────────────
	s.auditPublish(publishResult.ListingID, req)

	return publishResult, nil, nil
}

// auditPublish dispatches a platform audit event for a successful publish.
func (s *Service) auditPublish(listingID string, req PublishRequest) {
	audit.GlobalDispatch(audit.DataFusionAuditEvent{
		TenantID:   req.TenantID.String(),
		Action:     "marketplace.published",
		EntityType: "marketplace_listings",
		EntityID:   listingID,
		UserID:     req.ActorID.String(),
		AfterState: map[string]any{
			"title":            req.Title,
			"kind":             req.Kind,
			"artifact_version": req.Artifact.ArtifactVersion,
		},
	})
}

// InstallRequest is the domain-level input for installing a listing.
type InstallRequest struct {
	ListingID      uuid.UUID
	ActorID       uuid.UUID
	TenantID      uuid.UUID
	IdempotencyKey string
	RequestHash   []byte
}

// InstallResult is returned on a successful install.
type InstallResult struct {
	ListingID    string `json:"listing_id"`
	ArtifactType string `json:"artifact_type"`
	ArtifactKey  string `json:"artifact_key"`
	Version      string `json:"artifact_version"`
	IsNew        bool   `json:"is_new"` // true if this tenant had not installed before
}

// Install records a template subscription for the tenant.
// It does NOT write to bp_roles — the RBAC engine resolves the role_key
// via Gold Copy inheritance automatically.
func (s *Service) Install(ctx context.Context, req InstallRequest) (*InstallResult, *IdempotencyRecord, error) {
	repo := NewRepository(s.db)

	// ── 1. Idempotency ────────────────────────────────────────────────────
	if req.IdempotencyKey != "" {
		rec, found, err := repo.CheckIdempotency(ctx, req.IdempotencyKey,
			req.TenantID.String(), "install")
		if err != nil {
			return nil, nil, fmt.Errorf("idempotency check: %w", err)
		}
		if found {
			if string(rec.RequestHash) != string(req.RequestHash) {
				return nil, rec, fmt.Errorf("idempotency key reused with different payload")
			}
			var result InstallResult
			if err := json.Unmarshal(rec.ResponseBody, &result); err != nil {
				return nil, nil, fmt.Errorf("unmarshal cached response: %w", err)
			}
			return &result, rec, nil
		}
	}

	// ── 2. Fetch listing + latest artifact ─────────────────────────────────
	listing, err := repo.GetListingByID(ctx, req.ListingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("listing %s not found", req.ListingID)
		}
		return nil, nil, fmt.Errorf("get listing: %w", err)
	}

	if listing.Status != string(StatusPublished) {
		return nil, nil, fmt.Errorf("listing is not published (status=%s)", listing.Status)
	}

	// Parse artifact payload and validate.
	var artifact Artifact
	if listing.Payload != nil && *listing.Payload != "" {
		if err := json.Unmarshal([]byte(*listing.Payload), &artifact); err != nil {
			return nil, nil, fmt.Errorf("parse artifact payload: %w", err)
		}
		if verrs := artifact.Validate(); len(verrs) > 0 {
			return nil, nil, fmt.Errorf("artifact validation failed: %v", verrs)
		}
	}

	artifactKey := ""
	if v, ok := artifact.Definition["role_key"].(string); ok {
		artifactKey = v
	}
	artifactVersion := ""
	if listing.ArtifactVersion != nil {
		artifactVersion = *listing.ArtifactVersion
	}
	artifactType := "rbac_role"
	if listing.ArtifactType != nil {
		artifactType = *listing.ArtifactType
	}

	// ── 3. Transaction: install record + idempotency ──────────────────────
	var installResult *InstallResult
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	installRec := InstallRecord{
		TenantID:           req.TenantID,
		ListingID:          req.ListingID,
		ArtifactType:      artifactType,
		ArtifactKey:       artifactKey,
		ArtifactVersion:   artifactVersion,
		InstalledByActorID: req.ActorID,
	}

	isNew, err := RecordInstall(ctx, tx, installRec)
	if err != nil {
		return nil, nil, fmt.Errorf("record install: %w", err)
	}

	installResult = &InstallResult{
		ListingID:    req.ListingID.String(),
		ArtifactType: artifactType,
		ArtifactKey:  artifactKey,
		Version:      artifactVersion,
		IsNew:        isNew,
	}

	if req.IdempotencyKey != "" {
		body, _ := json.Marshal(installResult)
		rec := IdempotencyRecord{
			Key:            req.IdempotencyKey,
			TenantID:       req.TenantID.String(),
			Operation:      "install",
			RequestHash:    req.RequestHash,
			ResponseStatus: http.StatusOK,
			ResponseBody:   body,
		}
		if err := SaveIdempotency(ctx, tx, rec); err != nil {
			return nil, nil, fmt.Errorf("save idempotency: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}

	// ── 4. Audit ──────────────────────────────────────────────────────────
	audit.GlobalDispatch(audit.DataFusionAuditEvent{
		TenantID:   req.TenantID.String(),
		Action:     "marketplace.installed",
		EntityType: "installed_marketplace_artifacts",
		EntityID:   req.ListingID.String(),
		UserID:     req.ActorID.String(),
		AfterState: map[string]any{
			"artifact_type": artifactType,
			"artifact_key":  artifactKey,
			"is_new":       installResult.IsNew,
		},
	})

	return installResult, nil, nil
}

// BrowseResult is the API response shape for catalog browsing.
type BrowseResult struct {
	Listings []ListingResponse `json:"listings"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// ListingResponse is the public-facing shape of a marketplace listing.
type ListingResponse struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Kind               string  `json:"kind"`
	Category           string  `json:"category"`
	PublisherTenantID  string  `json:"publisher_tenant_id"`
	PublisherName      string  `json:"publisher_name"`
	Description        string  `json:"description"`
	PriceCents         int     `json:"price_cents"`
	BillingPeriod      string  `json:"billing_period"`
	Rating            float64 `json:"rating"`
	InstallsCount      int     `json:"installs_count"`
	ArtifactType       string  `json:"artifact_type,omitempty"`
	ArtifactVersion    string  `json:"artifact_version,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

// Browse returns a paginated, filtered catalog of published listings.
func (s *Service) Browse(ctx context.Context, params BrowseParams) (*BrowseResult, error) {
	repo := NewRepository(s.db)
	rows, total, err := repo.Browse(ctx, params)
	if err != nil {
		return nil, err
	}

	listings := make([]ListingResponse, 0, len(rows))
	for _, r := range rows {
		lr := ListingResponse{
			ID:                r.ID,
			Title:             r.Title,
			Kind:              r.Kind,
			Category:          r.Category,
			PublisherTenantID: r.PublisherTenantID,
			PublisherName:      r.PublisherName,
			Description:        r.Description,
			PriceCents:         r.PriceCents,
			BillingPeriod:      r.BillingPeriod,
			Rating:             r.Rating,
			InstallsCount:      r.InstallsCount,
			CreatedAt:          r.CreatedAt.Format(time.RFC3339),
		}
		if r.ArtifactType != nil {
			lr.ArtifactType = *r.ArtifactType
		}
		if r.ArtifactVersion != nil {
			lr.ArtifactVersion = *r.ArtifactVersion
		}
		listings = append(listings, lr)
	}

	return &BrowseResult{
		Listings: listings,
		Total:    total,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}, nil
}

// ProductEvolution returns AI-sourced recommendations for Gold Copy inclusion.
func (s *Service) ProductEvolution(ctx context.Context, minConfidence float64) ([]EvolutionRecommendation, error) {
	repo := NewRepository(s.db)
	rows, err := repo.GetProductEvolution(ctx, minConfidence)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil // table doesn't exist yet (Phase 2 not run)
	}

	recs := make([]EvolutionRecommendation, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, EvolutionRecommendation{
			ClusterID:       r.ClusterID,
			SampleName:     r.SampleName,
			EntityType:     r.EntityType,
			TenantCount:    r.TenantCount,
			Recommendation: fmt.Sprintf("Promote %q (used by %d tenants) to Gold Copy Core Master Schema",
				r.SampleName, r.TenantCount),
			ConfidenceScore: r.ConfidenceScore,
			DetectedAt:     r.DetectedAt.Format(time.RFC3339),
		})
	}
	return recs, nil
}

// GetInstallations returns all installed marketplace artifacts for a tenant.
func (s *Service) GetInstallations(ctx context.Context, tenantID uuid.UUID) ([]InstalledArtifact, error) {
	repo := NewRepository(s.db)
	return repo.GetInstallations(ctx, tenantID.String())
}

// EvolutionRecommendation is the API response for product-evolution.
type EvolutionRecommendation struct {
	ClusterID       string  `json:"cluster_id"`
	SampleName     string  `json:"sample_name"`
	EntityType     string  `json:"entity_type"`
	TenantCount    int     `json:"tenant_count"`
	Recommendation string  `json:"recommendation"`
	ConfidenceScore float64 `json:"confidence_score"`
	DetectedAt     string  `json:"detected_at"`
}

// MaxBodyBytes is the maximum request body size accepted for publish/install.
// Set to 1 MiB — the actual artifact definition inside is capped at 256 KiB by artifact.go.
const MaxBodyBytes = 1 << 20

// ReadAllBounds reads at most n bytes from r.  If limit is exceeded, returns
// an error BEFORE exhausting the reader, preventing memory exhaustion attacks.
func ReadAllBounds(r io.Reader, limit int) ([]byte, error) {
	lr := &io.LimitedReader{R: r, N: int64(limit) + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N == 0 {
		return nil, fmt.Errorf("request body exceeds maximum size of %d bytes", limit)
	}
	return data, nil
}
