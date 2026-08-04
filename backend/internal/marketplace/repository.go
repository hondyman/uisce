package marketplace

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles all persistence for marketplace entities.
// Non-transactional methods use r.db.  Transactional methods accept *sql.Tx
// from db.WithTenantTransaction.
type Repository struct {
	db *sqlx.DB
}

// NewRepository returns a Repository backed by db.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying db handle (for non-transactional queries).
func (r *Repository) DB() *sqlx.DB { return r.db }

// ListingRow is the raw database shape of a marketplace_listings row.
type ListingRow struct {
	ID                 string    `db:"id"`
	Title              string    `db:"title"`
	Kind               string    `db:"kind"`
	Category           string    `db:"category"`
	PublisherTenantID   string    `db:"publisher_tenant_id"`
	PublisherName      string    `db:"publisher_name"`
	PublisherActorID   *uuid.UUID `db:"publisher_actor_id"`
	Description        string    `db:"description"`
	PriceCents         int       `db:"price_cents"`
	BillingPeriod      string    `db:"billing_period"`
	Rating            float64   `db:"rating"`
	InstallsCount      int       `db:"installs_count"`
	Status            string    `db:"status"`
	VersionCount      int       `db:"version_count"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// BrowseParams encapsulates filter/sort/paginate inputs for catalog browsing.
type BrowseParams struct {
	Kind     string // rbac | rules_calculations | integration | bundle | "" (all)
	Category string // Compliance | ABAC | RBAC | Analytics | "" (all)
	Status   string // only published is supported for public browse
	Search   string // free-text match on title or description
	Limit    int    // max rows returned, capped at 100
	Offset   int
}

// ListingWithArtifact combines listing metadata with its latest artifact payload.
type ListingWithArtifact struct {
	ListingRow
	ArtifactID      *string `db:"artifact_id"`
	ArtifactType    *string `db:"artifact_type"`
	ArtifactVersion *string `db:"artifact_version"`
	Payload        *string `db:"payload"` // raw JSON string from DB
}

// Browse returns listings matching the filters, plus a total count for pagination.
func (r *Repository) Browse(ctx context.Context, p BrowseParams) ([]ListingWithArtifact, int, error) {
	conditions := []string{"ml.status = 'published'"}
	args := []any{}
	argIdx := 1

	if p.Kind != "" {
		conditions = append(conditions, fmt.Sprintf("ml.kind = $%d", argIdx))
		args = append(args, p.Kind)
		argIdx++
	}
	if p.Category != "" {
		conditions = append(conditions, fmt.Sprintf("ml.category = $%d", argIdx))
		args = append(args, p.Category)
		argIdx++
	}
	if p.Search != "" {
		conditions = append(conditions,
			fmt.Sprintf("(ml.title ILIKE $%d OR ml.description ILIKE $%d)", argIdx, argIdx+1))
		args = append(args, "%"+p.Search+"%", "%"+p.Search+"%")
		argIdx += 2
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	limit := p.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	countQuery := fmt.Sprintf(`SELECT count(*) FROM marketplace_listings ml %s`, where)
	var total int
	if err := sqlx.GetContext(ctx, r.db, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("browse count: %w", err)
	}

	args = append(args, limit, p.Offset)
	query := fmt.Sprintf(`
		SELECT
			ml.id, ml.title, ml.kind, ml.category, ml.publisher_tenant_id,
			ml.publisher_name, ml.publisher_actor_id, ml.description,
			ml.price_cents, ml.billing_period, ml.rating, ml.installs_count,
			ml.status, ml.version_count, ml.created_at, ml.updated_at,
			ma.artifact_id, ma.artifact_type, ma.artifact_version, ma.payload
		FROM marketplace_listings ml
		LEFT JOIN marketplace_artifacts ma
			ON ma.listing_id = ml.id AND ma.is_latest = TRUE
		%s
		ORDER BY ml.installs_count DESC, ml.rating DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	var rows []ListingWithArtifact
	if err := sqlx.SelectContext(ctx, r.db, &rows, query, args...); err != nil {
		return nil, 0, fmt.Errorf("browse select: %w", err)
	}
	return rows, total, nil
}

// GetListingByID returns a single listing with its latest artifact, or sql.ErrNoRows.
func (r *Repository) GetListingByID(ctx context.Context, listingID uuid.UUID) (*ListingWithArtifact, error) {
	query := `
		SELECT
			ml.id, ml.title, ml.kind, ml.category, ml.publisher_tenant_id,
			ml.publisher_name, ml.publisher_actor_id, ml.description,
			ml.price_cents, ml.billing_period, ml.rating, ml.installs_count,
			ml.status, ml.version_count, ml.created_at, ml.updated_at,
			ma.artifact_id, ma.artifact_type, ma.artifact_version, ma.payload
		FROM marketplace_listings ml
		LEFT JOIN marketplace_artifacts ma
			ON ma.listing_id = ml.id AND ma.is_latest = TRUE
		WHERE ml.id = $1`

	var row ListingWithArtifact
	if err := sqlx.GetContext(ctx, r.db, &row, query, listingID); err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateListing inserts a new marketplace_listings row inside an existing transaction.
func CreateListing(ctx context.Context, tx *sqlx.Tx, req CreateListingInput) (*ListingRow, error) {
	row := &ListingRow{}
	query := `
		INSERT INTO marketplace_listings
			(id, title, kind, category, publisher_tenant_id, publisher_name,
			 publisher_actor_id, description, price_cents, billing_period,
			 status, version_count)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 1)
		RETURNING
			id, title, kind, category, publisher_tenant_id, publisher_name,
			publisher_actor_id, description, price_cents, billing_period,
			rating, installs_count, status, version_count, created_at, updated_at`

	err := sqlx.GetContext(ctx, tx, row, query,
		req.ID, req.Title, req.Kind, req.Category,
		req.PublisherTenantID, req.PublisherName, req.PublisherActorID,
		req.Description, req.PriceCents, req.BillingPeriod, req.Status)
	if err != nil {
		return nil, fmt.Errorf("create listing: %w", err)
	}
	return row, nil
}

// CreateArtifact inserts a new artifact version and marks it as latest inside an existing transaction.
func CreateArtifact(ctx context.Context, tx *sqlx.Tx, listingID uuid.UUID,
	payload []byte, artifact Artifact) (*uuid.UUID, error) {

	var artifactID uuid.UUID
	insertQuery := `
		INSERT INTO marketplace_artifacts
			(listing_id, artifact_type, artifact_version, min_platform_version,
			 payload, canonical_sha256, is_latest)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		RETURNING id`

	err := sqlx.GetContext(ctx, tx, &artifactID, insertQuery,
		listingID, artifact.ArtifactType, artifact.ArtifactVersion,
		artifact.MinPlatformVersion, payload, artifact.CanonicalSHA256)
	if err != nil {
		return nil, fmt.Errorf("create artifact: %w", err)
	}

	// Demote previous versions.
	if _, err := tx.ExecContext(ctx, `
		UPDATE marketplace_artifacts
		SET is_latest = FALSE
		WHERE listing_id = $1 AND artifact_version != $2`,
		listingID, artifact.ArtifactVersion); err != nil {
		return nil, fmt.Errorf("demote previous versions: %w", err)
	}

	// Bump listing version_count.
	if _, err := tx.ExecContext(ctx, `
		UPDATE marketplace_listings SET version_count = version_count + 1 WHERE id = $1`,
		listingID); err != nil {
		return nil, fmt.Errorf("bump version_count: %w", err)
	}

	return &artifactID, nil
}

// RecordInstall inserts a subscription record inside an existing transaction.
func RecordInstall(ctx context.Context, tx *sqlx.Tx, req InstallRecord) (bool, error) {
	insertQuery := `
		INSERT INTO installed_marketplace_artifacts
			(tenant_id, listing_id, artifact_type, artifact_key,
			 artifact_version, installed_by_actor_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, listing_id) DO UPDATE SET
			artifact_version = EXCLUDED.artifact_version,
			installed_at = NOW(),
			installed_by_actor_id = EXCLUDED.installed_by_actor_id
		RETURNING id`

	var id uuid.UUID
	err := sqlx.GetContext(ctx, tx, &id, insertQuery,
		req.TenantID, req.ListingID, req.ArtifactType,
		req.ArtifactKey, req.ArtifactVersion, req.InstalledByActorID)
	if err != nil {
		return false, fmt.Errorf("record install: %w", err)
	}
	_ = id

	// Increment installs_count only when this is the first install for this listing by this tenant.
	// The CTE counts rows in installed_marketplace_artifacts where tenant has previously installed.
	result, err := tx.ExecContext(ctx, `
		UPDATE marketplace_listings ml
		SET installs_count = ml.installs_count +
			(SELECT CASE WHEN count(*) = 1 THEN 1 ELSE 0 END
			 FROM installed_marketplace_artifacts ima
			 WHERE ima.listing_id = ml.id AND ima.tenant_id = $1)
		WHERE ml.id = $2`,
		req.TenantID, req.ListingID)
	if err != nil {
		return false, fmt.Errorf("increment installs_count: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// IdempotencyRecord stores the result of a previously-processed operation.
type IdempotencyRecord struct {
	Key            string `db:"key"`
	TenantID      string `db:"tenant_id"`
	Operation     string `db:"operation"`
	RequestHash   []byte `db:"request_hash"`
	ResponseStatus int    `db:"response_status"`
	ResponseBody  []byte `db:"response_body"`
}

// CheckIdempotency looks up an existing idempotency key.
// Returns (record, true, nil) if found; (nil, false, nil) if not found.
func (r *Repository) CheckIdempotency(ctx context.Context, key, tenantID, operation string) (*IdempotencyRecord, bool, error) {
	query := `
		SELECT key, tenant_id, operation, request_hash,
			   response_status, response_body
		FROM marketplace_idempotency
		WHERE key = $1 AND tenant_id = $2 AND operation = $3
		  AND expires_at > NOW()`

	var rec IdempotencyRecord
	err := sqlx.GetContext(ctx, r.db, &rec, query, key, tenantID, operation)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("check idempotency: %w", err)
	}
	return &rec, true, nil
}

// SaveIdempotency persists an idempotency record inside an existing transaction.
func SaveIdempotency(ctx context.Context, tx *sqlx.Tx, rec IdempotencyRecord) error {
	query := `
		INSERT INTO marketplace_idempotency
			(key, tenant_id, operation, request_hash, response_status, response_body)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO UPDATE SET
			request_hash    = EXCLUDED.request_hash,
			response_status = EXCLUDED.response_status,
			response_body  = EXCLUDED.response_body,
			expires_at     = NOW() + INTERVAL '24 hours'`

	_, err := tx.ExecContext(ctx, query,
		rec.Key, rec.TenantID, rec.Operation,
		rec.RequestHash, rec.ResponseStatus, rec.ResponseBody)
	if err != nil {
		return fmt.Errorf("save idempotency: %w", err)
	}
	return nil
}

// GetInstallations returns all installed artifacts for a tenant (for the UI).
func (r *Repository) GetInstallations(ctx context.Context, tenantID string) ([]InstalledArtifact, error) {
	query := `
		SELECT
			ima.id, ima.listing_id, ima.artifact_type, ima.artifact_key,
			ima.artifact_version, ima.installed_at,
			ml.title AS listing_title, ml.category, ml.kind
		FROM installed_marketplace_artifacts ima
		JOIN marketplace_listings ml ON ml.id = ima.listing_id
		WHERE ima.tenant_id = $1
		ORDER BY ima.installed_at DESC`

	var rows []InstalledArtifact
	if err := sqlx.SelectContext(ctx, r.db, &rows, query, tenantID); err != nil {
		return nil, fmt.Errorf("get installations: %w", err)
	}
	return rows, nil
}

// InstalledArtifact is the read model for a tenant's installed marketplace artifacts.
type InstalledArtifact struct {
	ID              string    `db:"id"`
	ListingID       string    `db:"listing_id"`
	ArtifactType    string    `db:"artifact_type"`
	ArtifactKey     string    `db:"artifact_key"`
	ArtifactVersion string    `db:"artifact_version"`
	InstalledAt     time.Time `db:"installed_at"`
	ListingTitle    string    `db:"listing_title"`
	Category        string    `db:"category"`
	Kind            string    `db:"kind"`
}

// ProductEvolutionRow is a row from fact_customization_telemetry.
type ProductEvolutionRow struct {
	ID                 string    `db:"id"`
	ClusterID          string    `db:"cluster_id"`
	PatternHash        string    `db:"pattern_hash"`
	EntityType         string    `db:"entity_type"`
	SampleName          string    `db:"sample_name"`
	TenantCount         int       `db:"tenant_count"`
	RecommendedForCore  bool      `db:"recommended_for_core"`
	ConfidenceScore     float64   `db:"confidence_score"`
	DetectedAt          time.Time `db:"detected_at"`
}

// GetProductEvolution returns all recommendations ordered by tenant_count desc.
func (r *Repository) GetProductEvolution(ctx context.Context, minConfidence float64) ([]ProductEvolutionRow, error) {
	query := `
		SELECT id, cluster_id, pattern_hash, entity_type, sample_name,
			   tenant_count, recommended_for_core, confidence_score, detected_at
		FROM fact_customization_telemetry
		WHERE recommended_for_core = TRUE AND confidence_score >= $1
		ORDER BY tenant_count DESC`

	var rows []ProductEvolutionRow
	if err := sqlx.SelectContext(ctx, r.db, &rows, query, minConfidence); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get product evolution: %w", err)
	}
	return rows, nil
}

// CreateListingInput captures the fields needed to create a marketplace_listings row.
type CreateListingInput struct {
	ID                uuid.UUID
	Title             string
	Kind              string
	Category          string
	PublisherTenantID uuid.UUID
	PublisherName     string
	PublisherActorID  uuid.UUID
	Description       string
	PriceCents        int
	BillingPeriod     string
	Status            string
}

// InstallRecord captures the fields needed to record an installation.
type InstallRecord struct {
	TenantID          uuid.UUID
	ListingID         uuid.UUID
	ArtifactType      string
	ArtifactKey       string
	ArtifactVersion   string
	InstalledByActorID uuid.UUID
}

// MarshalPayload serialises an Artifact to canonical JSON bytes.
func MarshalPayload(a Artifact) ([]byte, error) {
	return json.Marshal(a)
}
