package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ExplorerQueryRequest is the simplified query object sent from the self-service UI.
type ExplorerQueryRequest struct {
	SavedID      *string   `json:"savedId,omitempty"`
	View         string    `json:"view"`
	DatasourceID string    `json:"datasource_id"`
	Region       string    `json:"region"`
	Measures     []string  `json:"measures"`
	Dimensions   []string  `json:"dimensions"`
	Filters      []Filter  `json:"filters"`
	Order        []OrderBy `json:"order"`
	Limit        *int      `json:"limit,omitempty"`
	Offset       *int      `json:"offset,omitempty"`
	Timezone     string    `json:"timezone"`
}

// Filter represents a single filter condition.
type Filter struct {
	Field  string   `json:"field"`
	Op     string   `json:"op"`
	Values []string `json:"values"`
}

// OrderBy specifies a sort order.
type OrderBy struct {
	Field string `json:"field"`
	Dir   string `json:"dir"`
}

// CompileResult is the response from the compile endpoint.
type CompileResult struct {
	SQL     string   `json:"sql"`
	GraphQL string   `json:"graphql"`
	Explain *Explain `json:"explain,omitempty"`
}

// Explain provides metadata about how the query was planned.
type Explain struct {
	UsedPreAgg       string `json:"used_preagg,omitempty"`
	RoutingReason    string `json:"routing_reason,omitempty"`
	RuleID           string `json:"rule_id,omitempty"`
	PartitionsPruned *int   `json:"partitions_pruned,omitempty"`
	Freshness        string `json:"freshness,omitempty"`
}

// ExecuteResult is the response from the execute endpoint.
type ExecuteResult struct {
	Columns            []ExplorerColumn `json:"columns"`
	Rows               []map[string]any `json:"rows"`
	Page               PageInfo         `json:"page"`
	DurationMs         int64            `json:"duration_ms"`
	UsedPreAggregation string           `json:"used_preaggregation,omitempty"`
	SQL                string           `json:"sql"`
	GraphQL            string           `json:"graphql"`
	Explain            *Explain         `json:"explain,omitempty"`
}

// ExplorerColumn describes a single column in the result set for the explorer API.
type ExplorerColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PageInfo contains pagination metadata.
type PageInfo struct {
	Limit      *int   `json:"limit,omitempty"`
	Offset     *int   `json:"offset,omitempty"`
	HasNext    bool   `json:"has_next"`
	TotalCount *int64 `json:"total_count,omitempty"` // Only if requested
}

// DataDomain represents a hierarchical domain (supports up to 3 levels).
type DataDomain struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Slug        string         `db:"slug" json:"slug"`
	ParentID    *uuid.UUID     `db:"parent_id" json:"parent_id,omitempty"`
	Level       int            `db:"level" json:"level"`
	Description sql.NullString `db:"description" json:"description,omitempty"`
	CreatedBy   sql.NullString `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

// ViewMemberDescription describes a dimension or measure exposed in a view.
type ViewMemberDescription struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// ViewDescription is the public contract for a view in the explorer.
type ViewDescription struct {
	Name        string                  `json:"name"`
	Schema      string                  `json:"schema"`
	Description string                  `json:"description"`
	Dimensions  []ViewMemberDescription `json:"dimensions"`
	Measures    []ViewMemberDescription `json:"measures"`
	Tags        []string                `json:"tags,omitempty"`
	Owner       string                  `json:"owner,omitempty"`
	Certified   bool                    `json:"certified,omitempty"`
}

// ViewMetadataDetails provides rich metadata for a single view.
type ViewMetadataDetails struct {
	Owner          string   `json:"owner"`
	Certified      bool     `json:"certified"`
	Freshness      string   `json:"freshness"`
	LastRefreshAgo string   `json:"lastRefreshAgo"`
	RunCount30d    int      `json:"runCount30d"`
	ExportGB30d    float64  `json:"exportGB30d"`
	Lineage        *Lineage `json:"lineage,omitempty"`
}

// Lineage represents the data lineage graph for a view.
type Lineage struct {
	Nodes []map[string]interface{} `json:"nodes"`
	Edges []map[string]interface{} `json:"edges"`
}

// ExplorerSavedQuery represents a query saved by a user in the explorer.
type ExplorerSavedQuery struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
	OwnerUserID    string          `db:"owner_user_id" json:"owner_user_id"`
	OwnerTenantID  string          `db:"owner_tenant_id" json:"owner_tenant_id"`
	Name           string          `db:"name" json:"name"`
	Description    sql.NullString  `db:"description" json:"description,omitempty"`
	PreviewDiff    json.RawMessage `db:"preview_diff" json:"preview_diff,omitempty"`
	Tags           pq.StringArray  `db:"tags" json:"tags,omitempty"`
	ViewName       string          `db:"view_name" json:"view_name"`
	Query          json.RawMessage `db:"query" json:"query"`
	VizConfig      json.RawMessage `db:"viz_config" json:"viz_config,omitempty"`
	Preview        json.RawMessage `db:"preview" json:"preview,omitempty"`
	LastRunAt      *time.Time      `db:"last_run_at" json:"last_run_at,omitempty"`
	LastDurationMs *int            `db:"last_duration_ms" json:"last_duration_ms,omitempty"`
	LastRowCount   *int64          `db:"last_row_count" json:"last_row_count,omitempty"`
	IsDeleted      bool            `db:"is_deleted" json:"is_deleted"`
}

// SavedQueryCreateRequest is the payload for creating a new saved query.
type SavedQueryCreateRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description *string         `json:"description"`
	ViewName    string          `json:"view_name" binding:"required"`
	Query       json.RawMessage `json:"query" binding:"required"`
	VizConfig   json.RawMessage `json:"viz_config"`
}

// ListSavedQueriesItem is the response item for the saved queries list endpoint.
type ListSavedQueriesItem struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	Name             string         `db:"name" json:"name"`
	ViewName         string         `db:"view_name" json:"view_name"`
	Tags             pq.StringArray `db:"tags" json:"tags,omitempty"`
	OwnerUserID      string         `db:"owner_user_id" json:"owner_user_id"`
	LastRunAt        *time.Time     `db:"last_run_at" json:"last_run_at,omitempty"`
	LastDurationMs   *int           `db:"last_duration_ms" json:"last_duration_ms,omitempty"`
	LastRowCount     *int64         `db:"last_row_count" json:"last_row_count,omitempty"`
	PreviewAvailable bool           `db:"preview_available" json:"preview_available"`
}

// SavedQueryACL represents an access control entry for a saved query.
type SavedQueryACL struct {
	ID           uuid.UUID `db:"id" json:"id"`
	SavedQueryID uuid.UUID `db:"saved_query_id" json:"saved_query_id"`
	SubjectType  string    `db:"subject_type" json:"subject_type"` // 'user', 'role', 'tenant'
	SubjectID    string    `db:"subject_id" json:"subject_id"`
	Permission   string    `db:"permission" json:"permission"` // 'read', 'write'
	GrantedBy    string    `db:"granted_by" json:"granted_by"`
	GrantedAt    time.Time `db:"granted_at" json:"granted_at"`
}

type ShareRequest struct {
	SubjectType string `json:"subject_type" binding:"required,oneof=user role tenant"`
	SubjectID   string `json:"subject_id" binding:"required"`
	Permission  string `json:"permission" binding:"required,oneof=read write"`
}

// SavedQueryUpdateRequest is the payload for updating an existing saved query.
type SavedQueryUpdateRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description *string         `json:"description"`
	Query       json.RawMessage `json:"query" binding:"required"`
	VizConfig   json.RawMessage `json:"viz_config"`
}

// Workbook represents a collection of saved tabs.
type Workbook struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description sql.NullString `db:"description" json:"description,omitempty"`
	OwnerUserID string         `db:"owner_user_id" json:"owner_user_id"`
	Tags        pq.StringArray `db:"tags" json:"tags,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

// WorkbookTab represents a single tab within a workbook.
type WorkbookTab struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	WorkbookID uuid.UUID       `db:"workbook_id" json:"workbook_id"`
	Title      string          `db:"title" json:"title"`
	ViewName   string          `db:"view_name" json:"view_name"`
	Query      json.RawMessage `db:"query" json:"query"`
	VizConfig  json.RawMessage `db:"viz_config" json:"viz_config,omitempty"`
	Position   int             `db:"position" json:"position"`
}

// FullWorkbook includes the workbook details and its tabs.
type FullWorkbook struct {
	Workbook
	Tabs []WorkbookTab `json:"tabs"`
}

// CreateWorkbookRequest is the payload for creating a new workbook.
type CreateWorkbookRequest struct {
	Name        string        `json:"name" binding:"required"`
	Description *string       `json:"description"`
	Tags        []string      `json:"tags"`
	Tabs        []WorkbookTab `json:"tabs" binding:"required,dive"`
}

// SuggestedQuery represents a query suggested for a particular view.
type SuggestedQuery struct {
	SavedQueryID uuid.UUID `db:"saved_query_id" json:"saved_query_id"`
	ViewName     string    `db:"view_name" json:"view_name"`
	Score        float64   `db:"score" json:"score"`
	Reason       string    `db:"reason" json:"reason"` // e.g., "Popular", "Team favorite"
	// We can join to get the query name etc.
	Name string `db:"name" json:"name"`
}

// ExplorerSavedQueryRun stores a snapshot of a single execution of a saved query.
type ExplorerSavedQueryRun struct {
	ID           uuid.UUID       `db:"id"`
	SavedQueryID uuid.UUID       `db:"saved_query_id"`
	ExecutedAt   time.Time       `db:"executed_at"`
	QueryHash    string          `db:"query_hash"`
	RowCount     int64           `db:"row_count"`
	Columns      pq.StringArray  `db:"columns"`
	Filters      json.RawMessage `db:"filters"`
}

// PreviewDiff represents the computed difference between two query runs.
type PreviewDiff struct {
	RowCount struct {
		Before int64 `json:"before"`
		After  int64 `json:"after"`
	} `json:"row_count"`
	Columns struct {
		Added   []string `json:"added"`
		Removed []string `json:"removed"`
	} `json:"columns"`
	// Filter diffing can be complex; this is a simplified version.
	FiltersChanged bool `json:"filters_changed"`
}

// Folder represents a container for saved queries and workbooks.
type Folder struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description sql.NullString `db:"description" json:"description,omitempty"`
	OwnerUserID string         `db:"owner_user_id" json:"owner_user_id"`
	ScopeType   sql.NullString `db:"scope_type" json:"scope_type,omitempty"` // 'user', 'role', 'tenant'
	ScopeID     sql.NullString `db:"scope_id" json:"scope_id,omitempty"`
	Tags        pq.StringArray `db:"tags" json:"tags,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

// FolderItem represents an item (query or workbook) within a folder.
type FolderItem struct {
	ID       uuid.UUID `db:"id" json:"id"`
	FolderID uuid.UUID `db:"folder_id" json:"folder_id"`
	ItemType string    `db:"item_type" json:"item_type"` // 'query' or 'workbook'
	ItemID   uuid.UUID `db:"item_id" json:"item_id"`
	Position int       `db:"position" json:"position"`
}

// FullFolder includes the folder details and its items.
type FullFolder struct {
	Folder
	Items []FolderItemDetail `json:"items"`
}

// FolderItemDetail includes the item's type and ID, plus its name for display.
type FolderItemDetail struct {
	ItemType string    `db:"item_type" json:"item_type"`
	ItemID   uuid.UUID `db:"item_id" json:"item_id"`
	Name     string    `db:"name" json:"name"`
	Position int       `db:"position" json:"position"`
}

// AddItemToFolderRequest is the payload for adding an item to a folder.
type AddItemToFolderRequest struct {
	ItemType string `json:"item_type" binding:"required,oneof=query workbook"`
	ItemID   string `json:"item_id" binding:"required"`
}

// SemanticSearchRequest is the payload for the semantic search endpoint.
type SemanticSearchRequest struct {
	Query        string `json:"query" binding:"required"`
	DatasourceID string `json:"datasource_id"`
	Region       string `json:"region"`
	Filters      struct {
		Type  []string `json:"type"`  // "query", "workbook"
		Scope string   `json:"scope"` // "mine", "shared", "all"
		Tags  []string `json:"tags"`
	} `json:"filters"`
}

// SemanticSearchResultItem represents a single item in the semantic search results.
type SemanticSearchResultItem struct {
	Type               string          `json:"type"` // "query" or "workbook"
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	Description        sql.NullString  `json:"description,omitempty"`
	Score              float64         `json:"score"`
	SemanticSimilarity float64         `json:"semantic_similarity"`
	UsageScore         float64         `json:"usage_score"`
	Preview            json.RawMessage `json:"preview,omitempty"`
	OwnerUserID        string          `json:"owner_user_id"`
	MatchedConcepts    []string        `json:"matched_concepts,omitempty"` // For explainability
	SourceSummary      string          `json:"source_summary,omitempty"`   // For explainability
	Certified          bool            `json:"certified"`
	Popular            bool            `json:"popular"`
	Reason             string          `json:"reason,omitempty"` // For suggestions
	HasAccess          bool            `json:"has_access"`
	IsRestricted       bool            `json:"is_restricted"`
	ClaimMatchScore    float64         `json:"claim_match_score"`
	QualifiedPath      string          `json:"qualified_path,omitempty"`
}

// SearchFeedbackRequest is the payload for logging feedback.
type SearchFeedbackRequest struct {
	Query      string `json:"query" binding:"required"`
	ResultID   string `json:"result_id" binding:"required"`
	ResultType string `json:"result_type" binding:"required,oneof=query workbook view"`
	Action     string `json:"action" binding:"required,oneof=clicked rerun favorited ignored"`
}

// SearchFeedback represents a user interaction with a search result.
type SearchFeedback struct {
	ID         uuid.UUID `db:"id"`
	UserID     string    `db:"user_id"`
	Query      string    `db:"query"`
	ResultID   string    `db:"result_id"`
	ResultType string    `db:"result_type"`
	Action     string    `db:"action"`
	Timestamp  time.Time `db:"timestamp"`
}

// FolderAnalyticsSummary represents high-level usage stats for a folder.
type FolderAnalyticsSummary struct {
	RunCount30d    int       `db:"run_count_30d" json:"run_count_30d"`
	ExportCount30d int       `db:"export_count_30d" json:"export_count_30d"`
	ViewerCount30d int       `db:"viewer_count_30d" json:"viewer_count_30d"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// FolderChangeLog represents a single entry in a folder's activity feed.
type FolderChangeLog struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ChangedAt   time.Time `db:"changed_at" json:"changed_at"`
	Actor       string    `db:"actor" json:"actor"`
	Action      string    `db:"action" json:"action"`
	Description string    `db:"description" json:"description"`
}

// TopFolderItem represents a frequently used item within a folder.
type TopFolderItem struct {
	ItemID   uuid.UUID `db:"item_id" json:"item_id"`
	ItemType string    `db:"item_type" json:"item_type"`
	Name     string    `db:"name" json:"name"`
	RunCount int       `db:"run_count" json:"run_count"`
}

// DuplicateQueryCluster groups saved queries that are identical.
type DuplicateQueryCluster struct {
	Fingerprint string                 `json:"fingerprint"`
	Queries     []ListSavedQueriesItem `json:"queries"`
}

// QueryTemplateMeta is a lightweight representation of a query template for lists.
type QueryTemplateMeta struct {
	ID           uuid.UUID      `db:"id" json:"id"`
	Name         string         `db:"name" json:"name"`
	Description  string         `db:"description" json:"description"`
	SemanticView string         `db:"semantic_view" json:"semantic_view"`
	Tags         pq.StringArray `db:"tags" json:"tags"`
	Certified    bool           `db:"certified" json:"certified"`
}

// QueryTemplate represents a full query template with its structure.
type QueryTemplate struct {
	QueryTemplateMeta
	DefaultDimensions []string        `db:"default_dimensions" json:"default_dimensions"`
	DefaultMetrics    []string        `db:"default_metrics" json:"default_metrics"`
	RequiredFilters   json.RawMessage `db:"required_filters" json:"required_filters"` // e.g., [{"field": "date", "op": "last_7_days"}]
	OwnerUserID       string          `db:"owner_user_id" json:"owner_user_id"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

// Anomaly represents a detected statistical anomaly in a metric.
type Anomaly struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	DatasourceID  string          `db:"datasource_id" json:"datasource_id"`
	TableName     string          `db:"table_name" json:"table_name"`
	Metric        string          `db:"metric" json:"metric"`
	TimeGrain     string          `db:"time_grain" json:"time_grain"` // 'daily', 'weekly'
	Timestamp     time.Time       `db:"timestamp" json:"timestamp"`
	Value         float64         `db:"value" json:"value"`
	ExpectedRange json.RawMessage `db:"expected_range" json:"expected_range"` // e.g., {"min": 4.5, "max": 5.5}
	AnomalyType   string          `db:"anomaly_type" json:"anomaly_type"`     // 'spike', 'dip', 'outlier'
	Severity      string          `db:"severity" json:"severity"`             // 'low', 'medium', 'high'
	Explanation   string          `db:"explanation" json:"explanation"`
	DetectedAt    time.Time       `db:"detected_at" json:"detected_at"`
}

// LineageNode represents a single node in a lineage graph.
type LineageNode struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"` // 'metric', 'view', 'query', 'table', 'workbook'
	Label string         `json:"label"`
	Data  map[string]any `json:"data,omitempty"`
}

// LineageEdge represents a connection between two nodes in a lineage graph.
type LineageEdge struct {
	ID     string `json:"id,omitempty"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Type   string `json:"type,omitempty"`
}

// LineageGraphData is the structure for the lineage graph API response.
type LineageGraphData struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

// ImpactAnalysis represents the downstream impact of an asset.
type ImpactAnalysis struct {
	Queries    []ImpactedItem `json:"queries"`
	Workbooks  []ImpactedItem `json:"workbooks"`
	Dashboards []ImpactedItem `json:"dashboards"`
}

// ImpactedItem is a simplified representation of a downstream asset.
type ImpactedItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Comment represents a user comment on an asset.
type Comment struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	AssetID      uuid.UUID  `db:"asset_id" json:"asset_id"`
	AssetType    string     `db:"asset_type" json:"asset_type"` // 'query', 'workbook', 'tab'
	AuthorUserID string     `db:"author_user_id" json:"author_user_id"`
	Body         string     `db:"body" json:"body"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	Resolved     bool       `db:"resolved" json:"resolved"`
	ParentID     *uuid.UUID `db:"parent_id" json:"parent_id,omitempty"`
}

// Approval represents the approval state of an asset.
type Approval struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	AssetID     uuid.UUID  `db:"asset_id" json:"asset_id"`
	AssetType   string     `db:"asset_type" json:"asset_type"`
	Status      string     `db:"status" json:"status"` // 'draft', 'pending', 'approved', 'rejected'
	RequestedBy string     `db:"requested_by" json:"requested_by"`
	ReviewedBy  *string    `db:"reviewed_by" json:"reviewed_by,omitempty"`
	DecisionAt  *time.Time `db:"decision_at" json:"decision_at,omitempty"`
	Notes       *string    `db:"notes" json:"notes,omitempty"`
}

// Alert represents a proactive notification for a user.
type Alert struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UserID      string    `db:"user_id" json:"user_id"`
	AssetID     uuid.UUID `db:"asset_id" json:"asset_id"`
	AssetType   string    `db:"asset_type" json:"asset_type"` // 'metric', 'query', 'goal'
	AlertType   string    `db:"alert_type" json:"alert_type"` // 'threshold_crossed', 'trend_change', 'new_match'
	Message     string    `db:"message" json:"message"`
	Severity    string    `db:"severity" json:"severity"` // 'info', 'warning', 'critical'
	TriggeredAt time.Time `db:"triggered_at" json:"triggered_at"`
	Read        bool      `db:"read" json:"read"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	IsRead      bool      `db:"is_read" json:"is_read"`
}

// DashboardSnapshot represents a point-in-time version of a dashboard.
type DashboardSnapshot struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	DashboardID     uuid.UUID       `db:"dashboard_id" json:"dashboard_id"`
	Name            string          `db:"name" json:"name"`
	Timestamp       time.Time       `db:"timestamp" json:"timestamp"`
	CreatedBy       string          `db:"created_by" json:"created_by"`
	Filters         json.RawMessage `db:"filters" json:"filters"`
	Layout          json.RawMessage `db:"layout" json:"layout"`
	Metrics         json.RawMessage `db:"metrics" json:"metrics"`
	Annotations     json.RawMessage `db:"annotations" json:"annotations"`
	SemanticContext json.RawMessage `db:"semantic_context" json:"semantic_context"`
	Certified       bool            `db:"certified" json:"certified"`
}

// SnapshotDiffItem represents a change in a diff.
type SnapshotDiffItem struct {
	Field      string `json:"field"`
	Before     string `json:"before"`
	After      string `json:"after"`
	ChangeType string `json:"change_type"` // 'added', 'removed', 'modified'
}

// SnapshotDiff represents the difference between two snapshots.
type SnapshotDiff struct {
	FiltersDiff  []SnapshotDiffItem `json:"filters_diff"`
	MetricsDiff  []SnapshotDiffItem `json:"metrics_diff"`
	LayoutDiff   []SnapshotDiffItem `json:"layout_diff"`
	SemanticDiff []SnapshotDiffItem `json:"semantic_diff"`
}

// SemanticViewVersion represents a single version of a semantic view.
type SemanticViewVersion struct {
	Version     int       `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
}

// SemanticDiff represents the difference between two versions of a semantic view.
type SemanticDiff struct {
	FromVersion int              `json:"from_version"`
	ToVersion   int              `json:"to_version"`
	Dimensions  []MemberDiffItem `json:"dimensions"`
	Metrics     []MemberDiffItem `json:"metrics"`
}

// MemberDiffItem represents a change to a dimension or metric.
type MemberDiffItem struct {
	Name       string `json:"name"`
	ChangeType string `json:"change_type"` // 'added', 'removed', 'modified'
	Before     string `json:"before,omitempty"`
	After      string `json:"after,omitempty"`
}

// SemanticModelClaim represents a permission grant for a user on a semantic model.
type SemanticModelClaim struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	UserID           string         `db:"user_id" json:"user_id"`
	TenantID         uuid.UUID      `db:"tenant_id" json:"tenant_id"` // For explicit isolation
	ModelID          uuid.UUID      `db:"model_id" json:"model_id"`
	Permission       string         `db:"permission" json:"permission"`         // 'read', 'write'
	Scope            pq.StringArray `db:"scope" json:"scope,omitempty"`         // e.g., ['metrics', 'dimensions']
	GrantedBy        string         `db:"granted_by" json:"granted_by"`         // e.g., 'role:analyst', 'direct_grant', 'bundle:finance_bundle'
	SourceID         *uuid.UUID     `db:"source_id" json:"source_id,omitempty"` // ID of the role mapping, bundle, etc.
	GrantedAt        time.Time      `db:"granted_at" json:"granted_at"`
	ExpiresAt        *time.Time     `db:"expires_at" json:"expires_at,omitempty"`
	RenewalRequested *bool          `db:"renewal_requested" json:"renewal_requested,omitempty"`
	RenewedAt        *time.Time     `db:"renewed_at" json:"renewed_at,omitempty"`
	RevokedAt        *time.Time     `db:"revoked_at" json:"revoked_at,omitempty"`
	LastUsedAt       *time.Time     `db:"last_used_at" json:"last_used_at,omitempty"` // For drift detection
	Status           string         `db:"status" json:"status"`                       // 'active', 'expiring', 'renewal_requested', 'expired', 'revoked'
}

// SemanticModelAccessRequest represents a user's request for permissions.
type SemanticModelAccessRequest struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	UserID              string     `db:"user_id" json:"user_id"`
	ModelID             uuid.UUID  `db:"model_id" json:"model_id"`
	RequestedPermission string     `db:"requested_permission" json:"requested_permission"`
	Reason              string     `db:"reason" json:"reason"`
	Status              string     `db:"status" json:"status"` // 'pending', 'approved', 'rejected'
	ReviewerID          *string    `db:"reviewer_id" json:"reviewer_id,omitempty"`
	DecisionNotes       *string    `db:"decision_notes" json:"decision_notes,omitempty"`
	RequestedAt         time.Time  `db:"requested_at" json:"requested_at"`
	DecidedAt           *time.Time `db:"decided_at" json:"decided_at,omitempty"`
}

// SemanticModelRoleClaim represents a permission grant for a role on a semantic model.
type SemanticModelRoleClaim struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	Role        string         `db:"role" json:"role"`
	ModelID     uuid.UUID      `db:"model_id" json:"model_id"`
	Permissions pq.StringArray `db:"permissions" json:"permissions"` // e.g., ['read']
	Scope       pq.StringArray `db:"scope" json:"scope,omitempty"`   // e.g., ['metrics', 'dimensions']
	GrantedBy   string         `db:"granted_by" json:"granted_by"`
	GrantedAt   time.Time      `db:"granted_at" json:"granted_at"`
}

// ClaimSimulationRequest is the payload for the claim simulation endpoint.
type ClaimSimulationRequest struct {
	SimulateFor    string          `json:"simulate_for"` // user ID or role name
	IsRole         bool            `json:"is_role"`
	ProposedClaims []ProposedClaim `json:"proposed_claims"`
}

// ClaimSimulationResult is the output of a claim simulation.
type ClaimSimulationResult struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	SimulatedFor   string          `db:"simulated_for" json:"simulated_for"`
	SimulatedBy    string          `db:"simulated_by" json:"simulated_by"`
	ProposedClaims json.RawMessage `db:"proposed_claims" json:"proposed_claims"`
	AffectedModels []AffectedModel `json:"affected_models"`
	RiskFlags      []string        `json:"risk_flags"`
	SimulatedAt    time.Time       `db:"simulated_at" json:"simulated_at"`
}

// AffectedModel provides details on a model impacted by a claim change.
type AffectedModel struct {
	ModelID   uuid.UUID `json:"model_id"`
	ModelName string    `json:"model_name"`
	Change    string    `json:"change"` // 'gained_read', 'gained_write'
	Certified bool      `json:"certified"`
}

// AccessControlAuditLog represents a log entry for a governance action.
type AccessControlAuditLog struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Timestamp   time.Time       `db:"timestamp" json:"timestamp"`
	ActorUserID string          `db:"actor_user_id" json:"actor_user_id"`
	Action      string          `db:"action" json:"action"`           // e.g., 'claim_granted', 'request_approved', 'role_updated'
	TargetType  string          `db:"target_type" json:"target_type"` // 'user', 'role', 'model'
	TargetID    string          `db:"target_id" json:"target_id"`
	Details     json.RawMessage `db:"details" json:"details,omitempty"` // e.g., {"model_id": "...", "permission": "read"}
}

// GovernanceSnapshot represents a point-in-time summary of governance metrics.
type GovernanceSnapshot struct {
	ID                      uuid.UUID       `db:"id" json:"id"`
	Timestamp               time.Time       `db:"timestamp" json:"timestamp"`
	CertifiedModelCount     int             `db:"certified_model_count" json:"certified_model_count"`
	UnresolvedRequestCount  int             `db:"unresolved_request_count" json:"unresolved_request_count"`
	RiskyClaimCount         int             `db:"risky_claim_count" json:"risky_claim_count"`
	SemanticCoveragePercent float64         `db:"semantic_coverage_percent" json:"semantic_coverage_percent"`
	RecentEvents            json.RawMessage `db:"recent_events" json:"recent_events"` // JSONB of recent audit events
}

// IndexJob represents a single run of the search indexer.
type IndexJob struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	JobType        string     `db:"job_type" json:"job_type"` // 'full', 'incremental', 'claim-sync'
	StartedAt      time.Time  `db:"started_at" json:"started_at"`
	CompletedAt    *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	Status         string     `db:"status" json:"status"` // 'pending', 'running', 'success', 'failed'
	AffectedAssets int        `json:"affected_assets"`
	TriggeredBy    string     `db:"triggered_by" json:"triggered_by"`
}

// AssetFreshness represents the indexing status of a single asset.
type AssetFreshness struct {
	AssetID       uuid.UUID `db:"asset_id" json:"asset_id"`
	AssetType     string    `db:"asset_type" json:"asset_type"`
	AssetName     string    `json:"asset_name"`
	LastIndexedAt time.Time `db:"last_indexed_at" json:"last_indexed_at"`
	Certified     bool      `db:"certified" json:"certified"`
}

// IndexMonitorSnapshot is the data payload for the index monitor dashboard.
type IndexMonitorSnapshot struct {
	LastFullRefresh     time.Time        `json:"last_full_refresh"`
	CertifiedCoverage   float64          `json:"certified_coverage"`
	SemanticHealthScore float64          `json:"semantic_health_score"`
	RecentJobs          []IndexJob       `json:"recent_jobs"`
	StaleAssets         []AssetFreshness `json:"stale_assets"`
	UnindexedAssetCount int              `json:"unindexed_asset_count"`
	// New fields for health score breakdown
	ClaimAlignment    float64 `json:"claim_alignment"`
	UsageCoverage     float64 `json:"usage_coverage"`
	AuditCompleteness float64 `json:"audit_completeness"`
	RiskExposure      float64 `json:"risk_exposure"`
}

// ClaimLifecycleEvent represents a single event in a claim's history.
type ClaimLifecycleEvent struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ClaimID     uuid.UUID `db:"claim_id" json:"claim_id"`
	EventType   string    `db:"event_type" json:"event_type"` // 'granted', 'renewal_requested', 'renewed', 'revoked', 'expired'
	ActorUserID string    `db:"actor_user_id" json:"actor_user_id"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	Notes       *string   `db:"notes" json:"notes,omitempty"`
}

// ClaimLifecycleSnapshot is the data payload for the claim lifecycle dashboard.
type ClaimLifecycleSnapshot struct {
	ActiveCount           int                   `json:"active_count"`
	ExpiringSoonCount     int                   `json:"expiring_soon_count"`
	RenewalRequestedCount int                   `json:"renewal_requested_count"`
	ExpiredCount          int                   `json:"expired_count"`
	RevokedCount          int                   `json:"revoked_count"`
	RecentEvents          []ClaimLifecycleEvent `json:"recent_events"`
}

// AccessControlPolicy defines a rule for governing claims.
type AccessControlPolicy struct {
	ID                    uuid.UUID       `db:"id" json:"id"`
	PolicyID              string          `db:"policy_id" json:"policy_id"` // e.g., "finance_read_default"
	Scope                 string          `db:"scope" json:"scope"`         // e.g., "domain:finance"
	Role                  string          `db:"role" json:"role"`
	Permissions           pq.StringArray  `db:"permissions" json:"permissions"`
	DurationDays          int             `db:"duration_days" json:"duration_days"`
	RequiresCertification bool            `db:"requires_certification" json:"requires_certification"`
	MaxClaimsPerUser      int             `db:"max_claims_per_user" json:"max_claims_per_user"`
	ApprovalThreshold     int             `db:"approval_threshold" json:"approval_threshold"`
	RenewalConditions     json.RawMessage `db:"renewal_conditions" json:"renewal_conditions"` // JSONB
	CreatedAt             time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time       `db:"updated_at" json:"updated_at"`
}

// PolicySimulationResult represents the outcome of a policy change simulation.
type PolicySimulationResult struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	PolicyID       string          `db:"policy_id" json:"policy_id"`
	SimulatedBy    string          `db:"simulated_by" json:"simulated_by"`
	SimulatedAt    time.Time       `db:"simulated_at" json:"simulated_at"`
	AffectedClaims json.RawMessage `db:"affected_claims" json:"affected_claims"` // e.g., {"added": 5, "removed": 2, "modified": 10}
	AffectedUsers  pq.StringArray  `db:"affected_users" json:"affected_users"`
	AffectedAssets pq.StringArray  `db:"affected_assets" json:"affected_assets"` // UUIDs of models, dashboards etc.
	RiskFlags      pq.StringArray  `db:"risk_flags" json:"risk_flags"`
	Notes          *string         `db:"notes" json:"notes,omitempty"`
}

// ClaimAwareLineageNode represents a lineage node decorated with user access info.
type ClaimAwareLineageNode struct {
	LineageNode
	Visibility string `json:"visibility"` // 'full', 'partial', 'none'
	Reason     string `json:"reason,omitempty"`
}

// ClaimAwareLineageGraphData is the structure for the claim-aware lineage graph API response.
type ClaimAwareLineageGraphData struct {
	Nodes []ClaimAwareLineageNode `json:"nodes"`
	Edges []LineageEdge           `json:"edges"`
}

// SemanticNotification represents a single notification for a user.
type SemanticNotification struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	EventType     string          `db:"event_type" json:"event_type"` // 'claim_granted', 'metric_changed', 'certification_updated'
	AssetID       uuid.UUID       `db:"asset_id" json:"asset_id"`
	AssetType     string          `db:"asset_type" json:"asset_type"`
	Message       string          `db:"message" json:"message"`
	TriggeredBy   string          `db:"triggered_by" json:"triggered_by"`
	Timestamp     time.Time       `db:"timestamp" json:"timestamp"`
	IsRead        bool            `db:"is_read" json:"is_read"`
	RoutingRuleID *string         `json:"routing_rule_id,omitempty"`
	RoutingTrace  json.RawMessage `json:"routing_trace,omitempty"` // JSON of { rule_id, resolved_recipients, reason }
	Status        string          `db:"status" json:"status"`      // 'sent', 'suppressed', 'escalated', 'resolved'
}

// GrantClaimRequest is the payload for granting a new claim via the intelligence service.
type GrantClaimRequest struct {
	UserID     string     `json:"user_id" binding:"required"`
	TenantID   uuid.UUID  `json:"tenant_id" binding:"required"`
	ModelID    uuid.UUID  `json:"model_id" binding:"required"`
	Permission string     `json:"permission" binding:"required"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// EvaluateAccessRequest is the payload for the real-time evaluation endpoint.
type EvaluateAccessRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required"`
	AssetID  string `json:"asset_id" binding:"required"` // Can be a model ID, metric ID, etc.
	Action   string `json:"action" binding:"required"`   // e.g., 'read', 'query', 'view_definition'
}

// EvaluateAccessResponse is the decision from the evaluation engine.
type EvaluateAccessResponse struct {
	Decision     string    `json:"decision"` // 'allow', 'deny', 'partial'
	Reason       string    `json:"reason"`
	AllowedScope []string  `json:"allowed_scope,omitempty"` // For partial access
	DecisionID   uuid.UUID `json:"decision_id"`             // Link to the decision log/trace
}

// SimulatedClaim represents a temporary claim for a simulation.
type SimulatedClaim struct {
	ModelID    uuid.UUID `json:"model_id"`
	Permission string    `json:"permission"`
}

// SimulateAccessRequest is the payload for the access simulation endpoint.
type SimulateAccessRequest struct {
	UserID          string           `json:"user_id" binding:"required"`
	TenantID        string           `json:"tenant_id" binding:"required"`
	AssetID         string           `json:"asset_id" binding:"required"`
	Action          string           `json:"action" binding:"required"`
	SimulatedClaims []SimulatedClaim `json:"simulated_claims,omitempty"`
	// SimulateRoles []string      `json:"simulate_roles,omitempty"` // Future enhancement
}
