package reporting

import (
	"time"

	"github.com/google/uuid"
)

type RadarSeverity string

const (
	SeverityInfo     RadarSeverity = "INFO"
	SeverityWarning  RadarSeverity = "WARNING"
	SeverityCritical RadarSeverity = "CRITICAL"
)

type ReviewStatus string

const (
	ReviewStatusPendingApproval ReviewStatus = "PENDING_APPROVAL"
	ReviewStatusApproved        ReviewStatus = "APPROVED"
	ReviewStatusRejected        ReviewStatus = "REJECTED"
	ReviewStatusAutoApproved    ReviewStatus = "AUTO_APPROVED"
)

type VarianceRuleDef struct {
	RuleKey       string        `json:"ruleKey"`
	RuleName      string        `json:"ruleName"`
	FieldKey      string        `json:"fieldKey"`
	ThresholdType string        `json:"thresholdType"` // "BPS", "PERCENT", "ABSOLUTE_DIFF", "CATEGORICAL"
	ThresholdVal  float64       `json:"thresholdVal"`
	Severity      RadarSeverity `json:"severity"`
	Weight        float64       `json:"weight"`
}

type RedlineDifferenceItem struct {
	ItemKey          string        `json:"itemKey"`          // e.g. "total_nav", "fee_accrual", "pos_AAPL"
	DisplayName      string        `json:"displayName"`
	BaselineValue    interface{}   `json:"baselineValue"`    // Prior Period / T0
	CurrentValue     interface{}   `json:"currentValue"`     // Draft Run / T1
	VarianceAmount   float64       `json:"varianceAmount"`
	VarianceBps      float64       `json:"varianceBps"`
	VariancePct      float64       `json:"variancePct"`
	Severity         RadarSeverity `json:"severity"`
	BreachDiagnostic string        `json:"breachDiagnostic"`
	IsBreached       bool          `json:"isBreached"`
}

type StatementApprovalTicket struct {
	TicketID         uuid.UUID               `json:"ticketId"`
	TenantID         uuid.UUID               `json:"tenantId"`
	StatementID      string                  `json:"statementId"`
	PortfolioID      string                  `json:"portfolioId"`
	EffectiveDate    time.Time               `json:"effectiveDate"`
	MakerIdentity    string                  `json:"makerIdentity"`
	CheckerIdentity  *string                 `json:"checkerIdentity,omitempty"`
	Status           ReviewStatus            `json:"status"`
	AnomalyScore     float64                 `json:"anomalyScore"`
	TotalBreaches    int                     `json:"totalBreaches"`
	CriticalBreaches int                     `json:"criticalBreaches"`
	Differences      []RedlineDifferenceItem `json:"differences"`
	CheckerNotes     string                  `json:"checkerNotes,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
	DecidedAt        *time.Time              `json:"decidedAt,omitempty"`
}
