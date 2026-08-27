package pagedesigner

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RelationshipCardinality string

const (
	CardinalityOneToOne  RelationshipCardinality = "1:1"
	CardinalityOneToMany RelationshipCardinality = "1:N"
	CardinalityManyToMany RelationshipCardinality = "N:M"
)

type CRUDEntitlements struct {
	AllowCreate   bool     `json:"allowCreate"`
	AllowUpdate   bool     `json:"allowUpdate"`
	AllowDelete   bool     `json:"allowDelete"`
	RequiredRoles []string `json:"requiredRoles,omitempty"`
}

type AutoPageGenerationRequest struct {
	TenantID             uuid.UUID        `json:"tenantId"`
	RootBOKey            string           `json:"rootBoKey"`
	BindingID            string           `json:"bindingId,omitempty"`
	PageGroupTitle       string           `json:"pageGroupTitle"`
	LayoutTopology       string           `json:"layoutTopology"` // 'TABBED_BY_SUBTYPE' | 'SINGLE_SCROLL_PANE' | 'MASTER_DETAIL_SPLIT'
	IncludeSubtypes      []string         `json:"includeSubtypes"`
	IncludeRelationships []string         `json:"includeRelationships"`
	CRUDEntitlements     CRUDEntitlements `json:"crudEntitlements"`
}

type GridSpan struct {
	XS int `json:"xs"`
	SM int `json:"sm,omitempty"`
	MD int `json:"md"`
	LG int `json:"lg"`
}

type EventAction struct {
	TargetChannel     string `json:"targetChannel"`
	SourcePropertyKey string `json:"sourcePropertyKey"`
	ActionType        string `json:"actionType"` // 'SET_PARAMETER' | 'CLEAR_PARAMETER' | 'NAVIGATE' | 'TRIGGER_REFETCH' | 'LAUNCH_MODAL_FORM'
	TargetPageKey     string `json:"targetPageKey,omitempty"`
}

type WidgetEventConfig struct {
	OnRowSelect      []EventAction `json:"onRowSelect,omitempty"`
	OnRowDoubleClick []EventAction `json:"onRowDoubleClick,omitempty"`
	OnChartSelect    []EventAction `json:"onChartSelect,omitempty"`
	OnFormSubmit     []EventAction `json:"onFormSubmit,omitempty"`
}

type PageWidgetDef struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"` // 'BO_FORM' | 'BO_GRID' | 'QUERY_VISUALIZATION' | 'METRIC_CARD' | 'FILTER_BAR'
	Title            string             `json:"title"`
	BOKey            *string            `json:"boKey,omitempty"`
	QueryID          *string            `json:"queryId,omitempty"`
	GridSpan         GridSpan           `json:"gridSpan"`
	SubscribedParams []string           `json:"subscribedParams"`
	Events           *WidgetEventConfig `json:"events,omitempty"`
	Entitlements     *CRUDEntitlements  `json:"entitlements,omitempty"`
}

type PageSectionSpec struct {
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	Flow    string          `json:"flow"` // 'ROW' | 'COLUMN'
	Widgets []PageWidgetDef `json:"widgets"`
}

type PageTabSpec struct {
	TabID       string            `json:"tabId"`
	TabTitle    string            `json:"tabTitle"`
	SubtypeCode *string           `json:"subtypeCode,omitempty"` // nil = Core Root BO
	Sections    []PageSectionSpec `json:"sections"`
}

type PageGroupSpec struct {
	PageGroupID string        `json:"pageGroupId"`
	RootBOKey   string        `json:"rootBoKey"`
	Title       string        `json:"title"`
	Tabs        []PageTabSpec `json:"tabs"`
}

type AutoPageCompiler struct {
	db *sqlx.DB
}

func NewAutoPageCompiler(db *sqlx.DB) *AutoPageCompiler {
	return &AutoPageCompiler{db: db}
}

// CompilePageGroup builds complete metadata for tabbed/subtyped CRUD pages
func (c *AutoPageCompiler) CompilePageGroup(ctx context.Context, req *AutoPageGenerationRequest) (*PageGroupSpec, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required (Rule 7 Security Mandate)")
	}
	if req.RootBOKey == "" {
		return nil, fmt.Errorf("rootBoKey is required")
	}

	spec := &PageGroupSpec{
		PageGroupID: fmt.Sprintf("pg_%s_%s", req.RootBOKey, uuid.New().String()[:8]),
		RootBOKey:   req.RootBOKey,
		Title:       req.PageGroupTitle,
		Tabs:        make([]PageTabSpec, 0),
	}

	// 1. Build Base / Core Tab
	baseTab := c.buildTabSpec("core", "Overview (Core)", req.RootBOKey, nil, req)
	spec.Tabs = append(spec.Tabs, baseTab)

	// 2. Build Subtype Tabs if requested
	if req.LayoutTopology == "TABBED_BY_SUBTYPE" {
		for _, stCode := range req.IncludeSubtypes {
			stCodeCopy := stCode
			stTab := c.buildTabSpec(stCode, fmt.Sprintf("%s Subtype", formatTitle(stCode)), req.RootBOKey, &stCodeCopy, req)
			spec.Tabs = append(spec.Tabs, stTab)
		}
	} else if req.LayoutTopology == "SINGLE_SCROLL_PANE" {
		// Single scroll pane incorporates additional sections for selected subtypes
		for _, stCode := range req.IncludeSubtypes {
			stWidget := PageWidgetDef{
				ID:               fmt.Sprintf("w_form_st_%s", stCode),
				Type:             "BO_FORM",
				Title:            fmt.Sprintf("%s Subtype Details", formatTitle(stCode)),
				BOKey:            &req.RootBOKey,
				GridSpan:         GridSpan{XS: 12, MD: 12, LG: 12},
				SubscribedParams: []string{"selected_id"},
				Entitlements:     &req.CRUDEntitlements,
			}
			spec.Tabs[0].Sections = append(spec.Tabs[0].Sections, PageSectionSpec{
				ID:      fmt.Sprintf("sec_st_%s", stCode),
				Title:   fmt.Sprintf("%s Subtype", formatTitle(stCode)),
				Flow:    "COLUMN",
				Widgets: []PageWidgetDef{stWidget},
			})
		}
	}

	return spec, nil
}

func (c *AutoPageCompiler) buildTabSpec(
	tabID, tabTitle, boKey string,
	subtypeCode *string,
	req *AutoPageGenerationRequest,
) PageTabSpec {
	tab := PageTabSpec{
		TabID:       tabID,
		TabTitle:    tabTitle,
		SubtypeCode: subtypeCode,
		Sections:    make([]PageSectionSpec, 0),
	}

	// Top Section: 1:1 Master Form Card (CRUD Enabled)
	formWidget := PageWidgetDef{
		ID:               fmt.Sprintf("w_form_%s", tabID),
		Type:             "BO_FORM",
		Title:            fmt.Sprintf("%s Details", tabTitle),
		BOKey:            &boKey,
		GridSpan:         GridSpan{XS: 12, MD: 12, LG: 12},
		SubscribedParams: []string{"selected_id"},
		Entitlements:     &req.CRUDEntitlements,
		Events: &WidgetEventConfig{
			OnFormSubmit: []EventAction{
				{TargetChannel: "selected_id", SourcePropertyKey: "id", ActionType: "SET_PARAMETER"},
			},
		},
	}

	tab.Sections = append(tab.Sections, PageSectionSpec{
		ID:      fmt.Sprintf("sec_master_%s", tabID),
		Title:   "Master Entity Information",
		Flow:    "COLUMN",
		Widgets: []PageWidgetDef{formWidget},
	})

	// Bottom Section: 1:N Child Relationships -> Auto-generate Infinite Scroll Grids
	for _, relKey := range req.IncludeRelationships {
		relKeyCopy := relKey
		gridWidget := PageWidgetDef{
			ID:               fmt.Sprintf("w_grid_%s_%s", tabID, relKey),
			Type:             "BO_GRID",
			Title:            fmt.Sprintf("Associated %s", formatTitle(relKey)),
			BOKey:            &relKeyCopy,
			GridSpan:         GridSpan{XS: 12, MD: 12, LG: 12},
			SubscribedParams: []string{"selected_id"},
			Events: &WidgetEventConfig{
				OnRowSelect: []EventAction{
					{TargetChannel: "selected_child_id", SourcePropertyKey: "id", ActionType: "SET_PARAMETER"},
				},
				OnRowDoubleClick: []EventAction{
					{TargetChannel: "active_modal_record_id", SourcePropertyKey: "id", ActionType: "LAUNCH_MODAL_FORM"},
				},
			},
		}

		tab.Sections = append(tab.Sections, PageSectionSpec{
			ID:      fmt.Sprintf("sec_rel_%s", relKey),
			Title:   fmt.Sprintf("Related: %s", formatTitle(relKey)),
			Flow:    "COLUMN",
			Widgets: []PageWidgetDef{gridWidget},
		})
	}

	return tab
}

func formatTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	parts := []rune(s)
	if parts[0] >= 'a' && parts[0] <= 'z' {
		parts[0] -= 32
	}
	for i := 1; i < len(parts); i++ {
		if parts[i-1] == '_' || parts[i-1] == '-' {
			if parts[i] >= 'a' && parts[i] <= 'z' {
				parts[i] -= 32
			}
		}
	}
	res := string(parts)
	res = fmt.Sprintf("%s", res)
	return res
}
