package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
)

// ListMicroBundles returns all available micro-bundles.
func ListMicroBundles(ctx context.Context, db *sql.DB) ([]models.MicroBundle, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, description, claims, domain, created_by, created_at, version FROM micro_bundle`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bundles []models.MicroBundle
	for rows.Next() {
		var b models.MicroBundle
		var claimsRaw []byte
		var desc sql.NullString
		var createdBy sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &desc, &claimsRaw, &b.Domain, &createdBy, &b.CreatedAt, &b.Version); err != nil {
			return nil, err
		}
		if desc.Valid {
			b.Description = desc.String
		} else {
			b.Description = ""
		}
		if createdBy.Valid {
			b.CreatedBy = createdBy.String
		} else {
			b.CreatedBy = ""
		}
		json.Unmarshal(claimsRaw, &b.Claims)
		bundles = append(bundles, b)
	}
	return bundles, nil
}

// CreateMicroBundle inserts a new micro-bundle.
func CreateMicroBundle(ctx context.Context, db *sql.DB, b *models.MicroBundle) error {
	b.ID = uuid.New()
	b.CreatedAt = time.Now()
	claimsJSON, _ := json.Marshal(b.Claims)
	_, err := db.ExecContext(ctx, `INSERT INTO micro_bundle (id, name, description, claims, domain, created_by, created_at, version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, b.ID, b.Name, b.Description, claimsJSON, b.Domain, b.CreatedBy, b.CreatedAt, b.Version)
	return err
}

// GetMicroBundle returns a single micro-bundle by id.
func GetMicroBundle(ctx context.Context, db *sql.DB, id uuid.UUID) (models.MicroBundle, error) {
	var b models.MicroBundle
	var claimsRaw []byte
	row := db.QueryRowContext(ctx, `SELECT id, name, description, claims, domain, created_by, created_at, version FROM micro_bundle WHERE id = $1`, id)
	var desc sql.NullString
	var createdBy sql.NullString
	if err := row.Scan(&b.ID, &b.Name, &desc, &claimsRaw, &b.Domain, &createdBy, &b.CreatedAt, &b.Version); err != nil {
		return b, err
	}
	if desc.Valid {
		b.Description = desc.String
	} else {
		b.Description = ""
	}
	if createdBy.Valid {
		b.CreatedBy = createdBy.String
	} else {
		b.CreatedBy = ""
	}
	json.Unmarshal(claimsRaw, &b.Claims)
	return b, nil
}

// UpdateMicroBundle updates an existing micro-bundle.
func UpdateMicroBundle(ctx context.Context, db *sql.DB, id uuid.UUID, b *models.MicroBundle) error {
	// bump version for optimistic change (caller may set Version)
	b.ID = id
	b.Version = b.Version + 1
	claimsJSON, _ := json.Marshal(b.Claims)
	_, err := db.ExecContext(ctx, `UPDATE micro_bundle SET name = $1, description = $2, claims = $3, domain = $4, version = $5 WHERE id = $6`, b.Name, b.Description, claimsJSON, b.Domain, b.Version, b.ID)
	return err
}

// DeleteMicroBundle removes a micro-bundle by id.
func DeleteMicroBundle(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	_, err := db.ExecContext(ctx, `DELETE FROM micro_bundle WHERE id = $1`, id)
	return err
}

// ListJITAddonGrants returns all JIT grants for a user.
func ListJITAddonGrants(ctx context.Context, db *sql.DB, userID string) ([]models.JITAddonGrant, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, user_id, bundle_id, granted_by, granted_at, expires_at, reason, status FROM jit_addon_grant WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []models.JITAddonGrant
	for rows.Next() {
		var g models.JITAddonGrant
		var grantedBy sql.NullString
		var reason sql.NullString
		if err := rows.Scan(&g.ID, &g.UserID, &g.BundleID, &grantedBy, &g.GrantedAt, &g.ExpiresAt, &reason, &g.Status); err != nil {
			return nil, err
		}
		if grantedBy.Valid {
			g.GrantedBy = grantedBy.String
		} else {
			g.GrantedBy = ""
		}
		if reason.Valid {
			g.Reason = reason.String
		} else {
			g.Reason = ""
		}
		grants = append(grants, g)
	}
	return grants, nil
}

// CreateJITAddonGrant inserts a new JIT add-on grant.
func CreateJITAddonGrant(ctx context.Context, db *sql.DB, g *models.JITAddonGrant) error {
	g.ID = uuid.New()
	g.GrantedAt = time.Now()
	_, err := db.ExecContext(ctx, `INSERT INTO jit_addon_grant (id, user_id, bundle_id, granted_by, granted_at, expires_at, reason, status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, g.ID, g.UserID, g.BundleID, g.GrantedBy, g.GrantedAt, g.ExpiresAt, g.Reason, g.Status)
	return err
}

// RenewJITAddonGrant updates the expiry of a JIT grant.
func RenewJITAddonGrant(ctx context.Context, db *sql.DB, grantID uuid.UUID, newExpiry time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE jit_addon_grant SET expires_at = $1, status = 'active' WHERE id = $2`, newExpiry, grantID)
	return err
}
