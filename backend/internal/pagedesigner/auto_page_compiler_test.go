package pagedesigner

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAutoPageCompiler_CompilePageGroup(t *testing.T) {
	compiler := NewAutoPageCompiler(nil)

	req := &AutoPageGenerationRequest{
		TenantID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		RootBOKey:            "account",
		PageGroupTitle:       "Account Master Studio",
		LayoutTopology:       "TABBED_BY_SUBTYPE",
		IncludeSubtypes:      []string{"institutional", "retail_wealth"},
		IncludeRelationships: []string{"positions", "mandate_info"},
		CRUDEntitlements: CRUDEntitlements{
			AllowCreate: true,
			AllowUpdate: true,
			AllowDelete: false,
		},
	}

	spec, err := compiler.CompilePageGroup(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, spec)

	// 1 Core tab + 2 Subtype tabs = 3 tabs
	assert.Equal(t, 3, len(spec.Tabs))
	assert.Equal(t, "core", spec.Tabs[0].TabID)
	assert.Equal(t, "institutional", spec.Tabs[1].TabID)
	assert.Equal(t, "retail_wealth", spec.Tabs[2].TabID)

	// In Core tab: 1 Master Form section + 2 Relationship Grid sections = 3 sections
	assert.Equal(t, 3, len(spec.Tabs[0].Sections))
	assert.Equal(t, "BO_FORM", spec.Tabs[0].Sections[0].Widgets[0].Type)
	assert.Equal(t, "BO_GRID", spec.Tabs[0].Sections[1].Widgets[0].Type)
	assert.Equal(t, "BO_GRID", spec.Tabs[0].Sections[2].Widgets[0].Type)

	// Double-click modal action on grids
	gridWidget := spec.Tabs[0].Sections[1].Widgets[0]
	assert.NotNil(t, gridWidget.Events)
	assert.Equal(t, 1, len(gridWidget.Events.OnRowDoubleClick))
	assert.Equal(t, "LAUNCH_MODAL_FORM", gridWidget.Events.OnRowDoubleClick[0].ActionType)
}
