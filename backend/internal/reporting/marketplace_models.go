package reporting

import (
	"time"

	"github.com/google/uuid"
)

type CloneReportRequest struct {
	TenantID        uuid.UUID  `json:"tenantId"`
	SourceReportID  uuid.UUID  `json:"sourceReportId"`
	TargetFolderID  *uuid.UUID `json:"targetFolderId,omitempty"`
	NewReportName   string     `json:"newReportName"`
	NewReportCode   string     `json:"newReportCode"`
	CopyPermissions bool       `json:"copyPermissions"`
}

type RebaseReportResult struct {
	ReportID        uuid.UUID `json:"reportId"`
	NewBaseVersion  int       `json:"newBaseVersion"`
	HasConflicts    bool      `json:"hasConflicts"`
	ConflictDetails []string  `json:"conflictDetails,omitempty"`
	AppliedPatches  int       `json:"appliedPatches"`
}

type NL2ReportPromptRequest struct {
	TenantID     uuid.UUID   `json:"tenantId"`
	UserPrompt   string      `json:"userPrompt"`
	TargetFormat PageFormat  `json:"targetFormat"`
	TargetPages  int         `json:"targetPages"`
	ContextBOIDs []uuid.UUID `json:"contextBoIds,omitempty"`
}

type MarketplacePackageSummary struct {
	PackageID     uuid.UUID `json:"packageId"`
	PackageCode   string    `json:"packageCode"`
	DisplayName   string    `json:"displayName"`
	PublisherName string    `json:"publisherName"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	Rating        float64   `json:"rating"`
	InstallCount  int       `json:"installCount"`
	IsVerified    bool      `json:"isVerified"`
	RequiredTerms []string  `json:"requiredTerms"`
	IsInstallable bool      `json:"isInstallable"`
	CreatedAt     time.Time `json:"createdAt"`
}
