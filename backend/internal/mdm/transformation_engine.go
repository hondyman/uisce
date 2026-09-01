package mdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DownstreamTargetPayload struct {
	TargetSystem    string                 `json:"target_system"`
	EntitySID       string                 `json:"entity_sid"`
	Payload         map[string]interface{} `json:"payload"`
	PayloadChecksum string                 `json:"payload_checksum"`
	DispatchedAt    time.Time              `json:"dispatched_at"`
}

type MDMTransformationEngine struct {
	db *sqlx.DB
}

type TransformationEngine = MDMTransformationEngine

func NewMDMTransformationEngine(db *sqlx.DB) *MDMTransformationEngine {
	return &MDMTransformationEngine{db: db}
}

func NewTransformationEngine(db *sqlx.DB) *MDMTransformationEngine {
	return NewMDMTransformationEngine(db)
}

// TransformRecord transforms a golden record for a specific binding ID
func (e *MDMTransformationEngine) TransformRecord(
	ctx context.Context,
	tenantID, bindingID uuid.UUID,
	goldAttributes map[string]interface{},
) (map[string]interface{}, string, error) {
	rawJSON, _ := json.Marshal(goldAttributes)
	hasher := sha256.New()
	hasher.Write(rawJSON)
	checksum := hex.EncodeToString(hasher.Sum(nil))
	return goldAttributes, checksum, nil
}


// TransformGoldenForTarget maps generic golden attributes into target-specific schemas and enums
func (e *MDMTransformationEngine) TransformGoldenForTarget(
	ctx context.Context,
	tenantID uuid.UUID,
	targetName string,
	entitySID string,
	goldenAttributes map[string]interface{},
) (*DownstreamTargetPayload, error) {
	transformed := make(map[string]interface{})

	switch targetName {
	case "CRIMS_ORACLE":
		if px, ok := goldenAttributes["market_price"]; ok {
			transformed["PX_LAST"] = px
		}
		if isin, ok := goldenAttributes["isin"]; ok {
			transformed["EXT_SEC_ID"] = isin
			transformed["SEC_ID_TYPE"] = "ISIN"
		}
		if ccy, ok := goldenAttributes["currency"]; ok {
			if ccy == "USD" {
				transformed["CURRENCY_CD"] = "USD"
			}
		}

	case "STARROCKS_LAKEHOUSE":
		for k, v := range goldenAttributes {
			transformed[k] = v
		}
		transformed["ingestion_time_tk"] = time.Now().UTC()
	}

	rawJSON, _ := json.Marshal(transformed)
	hasher := sha256.New()
	hasher.Write(rawJSON)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	return &DownstreamTargetPayload{
		TargetSystem:    targetName,
		EntitySID:       entitySID,
		Payload:         transformed,
		PayloadChecksum: checksum,
		DispatchedAt:    time.Now().UTC(),
	}, nil
}
