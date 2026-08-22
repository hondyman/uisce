package reporting

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DistributionDispatcher struct {
	db *sqlx.DB
}

func NewDistributionDispatcher(db *sqlx.DB) *DistributionDispatcher {
	return &DistributionDispatcher{db: db}
}

// DispatchBatchArtifacts routes generated reports to email, SFTP, and webhooks
func (d *DistributionDispatcher) DispatchBatchArtifacts(
	ctx context.Context,
	tenantID, batchID uuid.UUID,
) (int, int, error) {
	// Fetch all ready artifacts with their client distribution profiles
	query := `
		SELECT a.id AS artifact_id, a.client_id, a.storage_path, a.file_format,
		       c.channel_type, c.destination_target, c.encryption_type
		FROM public.report_burst_artifacts a
		JOIN public.client_distribution_configs c 
		  ON c.client_id = a.client_id AND c.tenant_id = a.tenant_id
		WHERE a.batch_id = $1 AND a.tenant_id = $2 AND a.status = 'READY' AND c.is_active = TRUE;
	`
	rows, err := d.db.QueryxContext(ctx, query, batchID, tenantID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	delivered := 0
	failed := 0

	for rows.Next() {
		var item struct {
			ArtifactID        uuid.UUID `db:"artifact_id"`
			ClientID          string    `db:"client_id"`
			StoragePath       string    `db:"storage_path"`
			FileFormat        string    `db:"file_format"`
			ChannelType       string    `db:"channel_type"`
			DestinationTarget string    `db:"destination_target"`
			EncryptionType    string    `db:"encryption_type"`
		}
		if err := rows.StructScan(&item); err != nil {
			continue
		}

		dispatchErr := d.deliverArtifact(ctx, item.ChannelType, item.DestinationTarget, item.StoragePath)
		status := "DELIVERED"
		errMsg := ""
		if dispatchErr != nil {
			status = "FAILED"
			errMsg = dispatchErr.Error()
			failed++
		} else {
			delivered++
		}

		// Log delivery audit entry
		_, _ = d.db.ExecContext(ctx, `
			INSERT INTO public.report_burst_distribution_logs (
				tenant_id, batch_id, artifact_id, client_id,
				channel_type, destination_target, delivery_status, error_message
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, tenantID, batchID, item.ArtifactID, item.ClientID, item.ChannelType, item.DestinationTarget, status, errMsg)
	}

	return delivered, failed, nil
}

func (d *DistributionDispatcher) deliverArtifact(ctx context.Context, channel, target, storagePath string) error {
	// In production, integrates with AWS SES (Email), SFTP SSH Client, or Signed S3 URL Webhook
	if channel == "EMAIL" {
		// Mock Email SMTP / SES Dispatch
		return nil
	} else if channel == "SFTP" {
		// Mock SFTP Stream upload
		return nil
	}
	return nil
}
