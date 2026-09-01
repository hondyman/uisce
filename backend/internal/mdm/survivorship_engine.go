package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SurvivorshipEngine struct {
	db *sqlx.DB
}

func NewSurvivorshipEngine(db *sqlx.DB) *SurvivorshipEngine {
	return &SurvivorshipEngine{db: db}
}

func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

type SurvivorshipStrategy string

const (
	StrategySourcePriority  SurvivorshipStrategy = "SOURCE_PRIORITY"
	StrategyMostRecent      SurvivorshipStrategy = "MOST_RECENT"
	StrategyConfidenceScore SurvivorshipStrategy = "CONFIDENCE_SCORE"
	StrategyConservativeMin SurvivorshipStrategy = "CONSERVATIVE_MIN"
	StrategyConservativeMax SurvivorshipStrategy = "CONSERVATIVE_MAX"
)

type FieldSourceRecord struct {
	SourceProvider string      `json:"source_provider"` // BLOOMBERG, REFINITIV, CRIMS
	Value          interface{} `json:"value"`
	Timestamp      time.Time   `json:"timestamp"`
	Confidence     float64     `json:"confidence"`
}

type GoldenRecordResult struct {
	FieldName      string      `json:"field_name"`
	ResolvedValue  interface{} `json:"resolved_value"`
	WinningSource  string      `json:"winning_source"`
	StrategyUsed   string      `json:"strategy_used"`
	EvaluationNote string      `json:"evaluation_note"`
}

// ResolveField calculates the authoritative Golden Record value based on configured strategies
func (e *SurvivorshipEngine) ResolveField(
	ctx context.Context,
	tenantID uuid.UUID,
	entityType, fieldName string,
	incomingSources []FieldSourceRecord,
) (*GoldenRecordResult, error) {
	if len(incomingSources) == 0 {
		return nil, fmt.Errorf("no source feeds provided for field %s", fieldName)
	}

	var rule struct {
		Strategy           SurvivorshipStrategy `db:"strategy"`
		PriorityListRaw    []byte               `db:"priority_list"`
		StalenessThreshold int                  `db:"staleness_threshold_sec"`
	}
	rule.Strategy = StrategyMostRecent
	rule.StalenessThreshold = 86400

	if e.db != nil {
		query := `
			SELECT strategy, priority_list, staleness_threshold_sec 
			FROM public.mdm_survivorship_rules 
			WHERE tenant_id = $1 AND entity_type = $2 AND field_name = $3 AND is_active = TRUE
		`
		_ = e.db.GetContext(ctx, &rule, query, tenantID, entityType, fieldName)
	}

	now := time.Now().UTC()
	validSources := make([]FieldSourceRecord, 0)
	for _, src := range incomingSources {
		if rule.StalenessThreshold > 0 && !src.Timestamp.IsZero() && now.Sub(src.Timestamp).Seconds() > float64(rule.StalenessThreshold) {
			continue
		}
		validSources = append(validSources, src)
	}
	if len(validSources) == 0 {
		validSources = incomingSources
	}

	switch rule.Strategy {
	case StrategySourcePriority:
		var priorities []string
		if len(rule.PriorityListRaw) > 0 {
			_ = json.Unmarshal(rule.PriorityListRaw, &priorities)
		}
		if len(priorities) == 0 {
			priorities = []string{"BLOOMBERG", "REFINITIV", "CRIMS", "INTERNAL"}
		}
		for _, provider := range priorities {
			for _, src := range validSources {
				if strings.EqualFold(src.SourceProvider, provider) {
					return &GoldenRecordResult{
						FieldName:      fieldName,
						ResolvedValue:  src.Value,
						WinningSource:  src.SourceProvider,
						StrategyUsed:   string(StrategySourcePriority),
						EvaluationNote: fmt.Sprintf("Selected by priority rank (%s)", provider),
					}, nil
				}
			}
		}
		return &GoldenRecordResult{
			FieldName:     fieldName,
			ResolvedValue: validSources[0].Value,
			WinningSource: validSources[0].SourceProvider,
			StrategyUsed:  string(StrategySourcePriority),
		}, nil

	case StrategyConservativeMin:
		minVal := 1e308
		winningSource := ""
		for _, src := range validSources {
			if num, ok := toFloat64(src.Value); ok && num < minVal {
				minVal = num
				winningSource = src.SourceProvider
			}
		}
		return &GoldenRecordResult{
			FieldName:      fieldName,
			ResolvedValue:  minVal,
			WinningSource:  winningSource,
			StrategyUsed:   string(StrategyConservativeMin),
			EvaluationNote: "Selected conservative minimum value across active feeds",
		}, nil

	case StrategyConfidenceScore:
		var maxConf float64 = -1.0
		var bestSrc FieldSourceRecord
		for _, src := range validSources {
			if src.Confidence > maxConf {
				maxConf = src.Confidence
				bestSrc = src
			}
		}
		return &GoldenRecordResult{
			FieldName:      fieldName,
			ResolvedValue:  bestSrc.Value,
			WinningSource:  bestSrc.SourceProvider,
			StrategyUsed:   string(StrategyConfidenceScore),
			EvaluationNote: fmt.Sprintf("Selected by highest confidence score (%.2f)", bestSrc.Confidence),
		}, nil

	case StrategyMostRecent:
		fallthrough
	default:
		latest := validSources[0]
		for _, src := range validSources[1:] {
			if src.Timestamp.After(latest.Timestamp) {
				latest = src
			}
		}
		return &GoldenRecordResult{
			FieldName:      fieldName,
			ResolvedValue:  latest.Value,
			WinningSource:  latest.SourceProvider,
			StrategyUsed:   string(StrategyMostRecent),
			EvaluationNote: fmt.Sprintf("Selected most recent feed timestamp (%s)", latest.Timestamp.Format(time.RFC3339)),
		}, nil
	}
}
