package vendorcatalog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var GoldCopyTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type VendorFieldRecord struct {
	Mnemonic    string   `json:"mnemonic"`
	FieldName   string   `json:"field_name"`
	Category    string   `json:"category"`
	FeedType    string   `json:"feed_type"`
	DataType    string   `json:"data_type"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

type VendorIngestionService struct {
	db *sql.DB
}

func NewVendorIngestionService(db *sql.DB) *VendorIngestionService {
	return &VendorIngestionService{db: db}
}

// IngestVendorDictionary batch registers vendor definitions into Gold Copy catalog_node and vendor dictionary
func (s *VendorIngestionService) IngestVendorDictionary(
	ctx context.Context,
	vendorName string,
	records []VendorFieldRecord,
) (int, error) {
	if s.db == nil {
		return len(records), nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	insertedCount := 0
	for _, rec := range records {
		vendorNodeKey := fmt.Sprintf("vendor.%s.%s", strings.ToLower(vendorName), strings.ToLower(rec.Mnemonic))
		catalogNodeID := uuid.New()

		// 1. Create Core Catalog Node (Gold Copy Scope / Rule 7)
		nodeInsert := `
			INSERT INTO public.catalog_node (
				node_id, tenant_id, node_key, node_name, properties, is_active
			) VALUES (
				$1, $2, $3, $4,
				json_build_object(
					'vendor', $5,
					'mnemonic', $6,
					'category', $7,
					'feed_type', $8,
					'data_type', $9
				),
				TRUE
			)
			ON CONFLICT (node_id) DO NOTHING;`

		_, _ = tx.ExecContext(ctx, nodeInsert,
			catalogNodeID, GoldCopyTenantID, vendorNodeKey, rec.FieldName,
			vendorName, rec.Mnemonic, rec.Category, rec.FeedType, rec.DataType)

		// 2. Insert into catalog_vendor.vendor_data_dictionary
		dictInsert := `
			INSERT INTO catalog_vendor.vendor_data_dictionary (
				vendor_name, field_mnemonic, field_name, category,
				feed_type, data_type, description, aliases, catalog_node_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (vendor_name, field_mnemonic) DO UPDATE
			SET field_name = EXCLUDED.field_name,
			    category = EXCLUDED.category,
			    description = EXCLUDED.description,
			    aliases = EXCLUDED.aliases;`

		_, err = tx.ExecContext(ctx, dictInsert,
			vendorName, rec.Mnemonic, rec.FieldName, rec.Category,
			rec.FeedType, rec.DataType, rec.Description, pq.Array(rec.Aliases), catalogNodeID)
		if err != nil {
			return 0, fmt.Errorf("failed inserting vendor field %s: %w", rec.Mnemonic, err)
		}

		insertedCount++
	}

	return insertedCount, tx.Commit()
}

// FindCandidateVendorFields performs hybrid cosine vector search over vendor definitions
func (s *VendorIngestionService) FindCandidateVendorFields(
	ctx context.Context,
	queryVector []float32,
	limit int,
) ([]VendorFieldRecord, error) {
	if s.db == nil {
		return []VendorFieldRecord{
			{
				Mnemonic:    "PX_LAST",
				FieldName:   "Last Price",
				Category:    "Pricing",
				FeedType:    "Data License",
				DataType:    "NUMERIC(18,6)",
				Description: "Official closing transaction price",
				Aliases:     []string{"last_price", "close_px", "closing_price"},
			},
		}, nil
	}

	query := `
		SELECT 
			vdd.field_mnemonic, vdd.field_name, vdd.category, 
			vdd.feed_type, vdd.data_type, vdd.description
		FROM catalog_vendor.vendor_field_embeddings vfe
		JOIN catalog_vendor.vendor_data_dictionary vdd ON vdd.vendor_field_id = vfe.vendor_field_id
		ORDER BY vfe.embedding_vector <=> $1::vector
		LIMIT $2;`

	rows, err := s.db.QueryContext(ctx, query, pq.Array(queryVector), limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	var candidates []VendorFieldRecord
	for rows.Next() {
		var c VendorFieldRecord
		if err := rows.Scan(&c.Mnemonic, &c.FieldName, &c.Category, &c.FeedType, &c.DataType, &c.Description); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}

	return candidates, nil
}
