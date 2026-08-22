package mdm

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BatchFileLoader struct {
	db       *sqlx.DB
	resolver *UniversalMDMResolver
}

func NewBatchFileLoader(db *sqlx.DB, resolver *UniversalMDMResolver) *BatchFileLoader {
	return &BatchFileLoader{
		db:       db,
		resolver: resolver,
	}
}

// IngestVendorFileStream streams bulk CSV/Parquet vendor drops directly into the MDM engine
func (l *BatchFileLoader) IngestVendorFileStream(
	ctx context.Context,
	tenantID uuid.UUID,
	domainKey, vendorName string,
	reader io.Reader,
) (int, int, error) {
	if tenantID == uuid.Nil {
		return 0, 0, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil") // Rule 7 Guard
	}

	csvReader := csv.NewReader(reader)
	csvReader.ReuseRecord = true

	// Read Header
	headers, err := csvReader.Read()
	if err != nil {
		return 0, 0, fmt.Errorf("failed reading CSV header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	batch := make([]VendorFeedPayload, 0, 1000)
	totalProcessed := 0
	successful := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		totalProcessed++
		attributes := make(map[string]interface{})
		masterSID := ""

		for fieldName, idx := range headerMap {
			val := record[idx]
			if fieldName == "security_sid" || fieldName == "master_sid" || fieldName == "isin" {
				masterSID = val
			}

			// Numeric float parsing
			if num, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
				attributes[fieldName] = num
			} else {
				attributes[fieldName] = val
			}
		}

		if masterSID == "" {
			masterSID = fmt.Sprintf("ENTITY_%d", totalProcessed)
		}

		batch = append(batch, VendorFeedPayload{
			DomainKey:       domainKey,
			MasterEntitySID: masterSID,
			VendorName:      vendorName,
			EffectiveDate:   time.Now().UTC(),
			Attributes:      attributes,
			ConfidenceScore: 0.95,
		})

		if len(batch) >= 1000 {
			// Process chunk
			for _, item := range batch {
				_, _, masterErr := l.resolver.MasterIncomingFeeds(ctx, tenantID, domainKey, item.MasterEntitySID, item.EffectiveDate, []VendorFeedPayload{item})
				if masterErr == nil {
					successful++
				}
			}
			batch = batch[:0]
		}
	}

	// Flush remainder
	for _, item := range batch {
		_, _, masterErr := l.resolver.MasterIncomingFeeds(ctx, tenantID, domainKey, item.MasterEntitySID, item.EffectiveDate, []VendorFeedPayload{item})
		if masterErr == nil {
			successful++
		}
	}

	return totalProcessed, successful, nil
}
