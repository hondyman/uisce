package semantic

import (
	"context"
	"fmt"
)

// WealthManagementCubes provides pre-configured cubes for wealth management
type WealthManagementCubes struct {
	service *Service
}

// NewWealthManagementCubes creates wealth management cube definitions
func NewWealthManagementCubes(service *Service) *WealthManagementCubes {
	return &WealthManagementCubes{service: service}
}

// InitializeWealthCubes creates all wealth management cubes for a tenant
func (wm *WealthManagementCubes) InitializeWealthCubes(ctx context.Context, tenantID string) error {
	cubes := []struct {
		cube       *Cube
		dimensions []Dimension
		measures   []Measure
	}{
		{
			cube:       wm.portfolioCube(tenantID),
			dimensions: wm.portfolioDimensions(),
			measures:   wm.portfolioMeasures(),
		},
		{
			cube:       wm.clientCube(tenantID),
			dimensions: wm.clientDimensions(),
			measures:   wm.clientMeasures(),
		},
		{
			cube:       wm.assetAllocationCube(tenantID),
			dimensions: wm.assetAllocationDimensions(),
			measures:   wm.assetAllocationMeasures(),
		},
		{
			cube:       wm.performanceCube(tenantID),
			dimensions: wm.performanceDimensions(),
			measures:   wm.performanceMeasures(),
		},
		{
			cube:       wm.advisorCube(tenantID),
			dimensions: wm.advisorDimensions(),
			measures:   wm.advisorMeasures(),
		},
	}

	for _, def := range cubes {
		// Create cube
		if err := wm.service.CreateCube(ctx, def.cube); err != nil {
			return fmt.Errorf("failed to create cube %s: %w", def.cube.Name, err)
		}

		// Create dimensions
		for _, dim := range def.dimensions {
			dim.CubeID = def.cube.ID
			if err := wm.service.CreateDimension(ctx, &dim); err != nil {
				return fmt.Errorf("failed to create dimension %s: %w", dim.Name, err)
			}
		}

		// Create measures
		for _, measure := range def.measures {
			measure.CubeID = def.cube.ID
			if err := wm.service.CreateMeasure(ctx, &measure); err != nil {
				return fmt.Errorf("failed to create measure %s: %w", measure.Name, err)
			}
		}
	}

	return nil
}

// Portfolio Cube
func (wm *WealthManagementCubes) portfolioCube(tenantID string) *Cube {
	return &Cube{
		TenantID:    tenantID,
		Name:        "portfolios",
		DisplayName: "Portfolios",
		Description: "Portfolio holdings and valuations",
		SQL: `
			SELECT 
				p.id as portfolio_id,
				p.client_id,
				p.name as portfolio_name,
				p.account_number,
				h.asset_id,
				h.quantity,
				h.cost_basis,
				h.current_value,
				h.unrealized_gain_loss,
				h.updated_at
			FROM portfolios p
			JOIN holdings h ON h.portfolio_id = p.id
			WHERE p.tenant_id = '` + tenantID + `'
		`,
		Status: "active",
	}
}

func (wm *WealthManagementCubes) portfolioDimensions() []Dimension {
	return []Dimension{
		{Name: "portfolio_id", DisplayName: "Portfolio ID", Type: "string", SQL: "portfolio_id", PrimaryKey: true},
		{Name: "client_id", DisplayName: "Client ID", Type: "string", SQL: "client_id"},
		{Name: "portfolio_name", DisplayName: "Portfolio Name", Type: "string", SQL: "portfolio_name"},
		{Name: "account_number", DisplayName: "Account Number", Type: "string", SQL: "account_number"},
		{Name: "asset_id", DisplayName: "Asset ID", Type: "string", SQL: "asset_id"},
		{Name: "updated_at", DisplayName: "Last Updated", Type: "time", SQL: "updated_at"},
	}
}

func (wm *WealthManagementCubes) portfolioMeasures() []Measure {
	return []Measure{
		{Name: "total_value", DisplayName: "Total Value", Type: "sum", SQL: "SUM(current_value)", Format: "currency"},
		{Name: "total_cost_basis", DisplayName: "Total Cost Basis", Type: "sum", SQL: "SUM(cost_basis)", Format: "currency"},
		{Name: "total_gain_loss", DisplayName: "Total Gain/Loss", Type: "sum", SQL: "SUM(unrealized_gain_loss)", Format: "currency"},
		{Name: "portfolio_count", DisplayName: "Portfolio Count", Type: "count", SQL: "COUNT(DISTINCT portfolio_id)"},
		{Name: "avg_portfolio_value", DisplayName: "Avg Portfolio Value", Type: "avg", SQL: "AVG(current_value)", Format: "currency"},
	}
}

// Client Cube
func (wm *WealthManagementCubes) clientCube(tenantID string) *Cube {
	return &Cube{
		TenantID:    tenantID,
		Name:        "clients",
		DisplayName: "Clients",
		Description: "Client demographics and relationships",
		SQL: `
			SELECT 
				c.id as client_id,
				c.name as client_name,
				c.email,
				c.risk_tolerance,
				c.advisor_id,
				c.client_type,
				c.created_at,
				c.updated_at,
				COALESCE(SUM(p.total_value), 0) as aum
			FROM clients c
			LEFT JOIN portfolios p ON p.client_id = c.id
			WHERE c.tenant_id = '` + tenantID + `'
			GROUP BY c.id, c.name, c.email, c.risk_tolerance, c.advisor_id, c.client_type, c.created_at, c.updated_at
		`,
		Status: "active",
	}
}

func (wm *WealthManagementCubes) clientDimensions() []Dimension {
	return []Dimension{
		{Name: "client_id", DisplayName: "Client ID", Type: "string", SQL: "client_id", PrimaryKey: true},
		{Name: "client_name", DisplayName: "Client Name", Type: "string", SQL: "client_name"},
		{Name: "email", DisplayName: "Email", Type: "string", SQL: "email"},
		{Name: "risk_tolerance", DisplayName: "Risk Tolerance", Type: "string", SQL: "risk_tolerance"},
		{Name: "advisor_id", DisplayName: "Advisor ID", Type: "string", SQL: "advisor_id"},
		{Name: "client_type", DisplayName: "Client Type", Type: "string", SQL: "client_type"},
		{Name: "created_at", DisplayName: "Onboarded Date", Type: "time", SQL: "created_at"},
	}
}

func (wm *WealthManagementCubes) clientMeasures() []Measure {
	return []Measure{
		{Name: "client_count", DisplayName: "Client Count", Type: "count", SQL: "COUNT(DISTINCT client_id)"},
		{Name: "total_aum", DisplayName: "Total AUM", Type: "sum", SQL: "SUM(aum)", Format: "currency"},
		{Name: "avg_aum", DisplayName: "Avg AUM per Client", Type: "avg", SQL: "AVG(aum)", Format: "currency"},
	}
}

// Asset Allocation Cube
func (wm *WealthManagementCubes) assetAllocationCube(tenantID string) *Cube {
	return &Cube{
		TenantID:    tenantID,
		Name:        "asset_allocation",
		DisplayName: "Asset Allocation",
		Description: "Asset class distribution and allocation",
		SQL: `
			SELECT 
				h.portfolio_id,
				a.asset_class,
				a.asset_type,
				a.sector,
				a.region,
				h.quantity,
				h.current_value,
				h.cost_basis
			FROM holdings h
			JOIN assets a ON a.id = h.asset_id
			JOIN portfolios p ON p.id = h.portfolio_id
			WHERE p.tenant_id = '` + tenantID + `'
		`,
		Status: "active",
	}
}

func (wm *WealthManagementCubes) assetAllocationDimensions() []Dimension {
	return []Dimension{
		{Name: "portfolio_id", DisplayName: "Portfolio ID", Type: "string", SQL: "portfolio_id"},
		{Name: "asset_class", DisplayName: "Asset Class", Type: "string", SQL: "asset_class"},
		{Name: "asset_type", DisplayName: "Asset Type", Type: "string", SQL: "asset_type"},
		{Name: "sector", DisplayName: "Sector", Type: "string", SQL: "sector"},
		{Name: "region", DisplayName: "Region", Type: "string", SQL: "region"},
	}
}

func (wm *WealthManagementCubes) assetAllocationMeasures() []Measure {
	return []Measure{
		{Name: "allocation_value", DisplayName: "Allocation Value", Type: "sum", SQL: "SUM(current_value)", Format: "currency"},
		{Name: "allocation_pct", DisplayName: "Allocation %", Type: "sum", SQL: "SUM(current_value) / NULLIF(SUM(SUM(current_value)) OVER (), 0) * 100", Format: "percent"},
		{Name: "position_count", DisplayName: "Position Count", Type: "count", SQL: "COUNT(*)"},
	}
}

// Performance Cube
func (wm *WealthManagementCubes) performanceCube(tenantID string) *Cube {
	return &Cube{
		TenantID:    tenantID,
		Name:        "performance",
		DisplayName: "Performance",
		Description: "Portfolio performance and returns",
		SQL: `
			SELECT 
				pr.portfolio_id,
				pr.date,
				pr.total_value,
				pr.daily_return,
				pr.mtd_return,
				pr.ytd_return,
				pr.inception_return,
				pr.benchmark_return
			FROM portfolio_returns pr
			JOIN portfolios p ON p.id = pr.portfolio_id
			WHERE p.tenant_id = '` + tenantID + `'
		`,
		Status: "active",
	}
}

func (wm *WealthManagementCubes) performanceDimensions() []Dimension {
	return []Dimension{
		{Name: "portfolio_id", DisplayName: "Portfolio ID", Type: "string", SQL: "portfolio_id"},
		{Name: "date", DisplayName: "Date", Type: "time", SQL: "date"},
	}
}

func (wm *WealthManagementCubes) performanceMeasures() []Measure {
	return []Measure{
		{Name: "total_value", DisplayName: "Total Value", Type: "sum", SQL: "SUM(total_value)", Format: "currency"},
		{Name: "avg_daily_return", DisplayName: "Avg Daily Return", Type: "avg", SQL: "AVG(daily_return)", Format: "percent"},
		{Name: "avg_mtd_return", DisplayName: "Avg MTD Return", Type: "avg", SQL: "AVG(mtd_return)", Format: "percent"},
		{Name: "avg_ytd_return", DisplayName: "Avg YTD Return", Type: "avg", SQL: "AVG(ytd_return)", Format: "percent"},
		{Name: "avg_inception_return", DisplayName: "Avg Inception Return", Type: "avg", SQL: "AVG(inception_return)", Format: "percent"},
		{Name: "avg_benchmark_return", DisplayName: "Avg Benchmark Return", Type: "avg", SQL: "AVG(benchmark_return)", Format: "percent"},
	}
}

// Advisor Cube
func (wm *WealthManagementCubes) advisorCube(tenantID string) *Cube {
	return &Cube{
		TenantID:    tenantID,
		Name:        "advisors",
		DisplayName: "Advisors",
		Description: "Advisor performance and client metrics",
		SQL: `
			SELECT 
				a.id as advisor_id,
				a.name as advisor_name,
				a.team,
				c.id as client_id,
				COALESCE(SUM(p.total_value), 0) as client_aum
			FROM advisors a
			LEFT JOIN clients c ON c.advisor_id = a.id
			LEFT JOIN portfolios p ON p.client_id = c.id
			WHERE a.tenant_id = '` + tenantID + `'
			GROUP BY a.id, a.name, a.team, c.id
		`,
		Status: "active",
	}
}

func (wm *WealthManagementCubes) advisorDimensions() []Dimension {
	return []Dimension{
		{Name: "advisor_id", DisplayName: "Advisor ID", Type: "string", SQL: "advisor_id", PrimaryKey: true},
		{Name: "advisor_name", DisplayName: "Advisor Name", Type: "string", SQL: "advisor_name"},
		{Name: "team", DisplayName: "Team", Type: "string", SQL: "team"},
	}
}

func (wm *WealthManagementCubes) advisorMeasures() []Measure {
	return []Measure{
		{Name: "advisor_count", DisplayName: "Advisor Count", Type: "count", SQL: "COUNT(DISTINCT advisor_id)"},
		{Name: "client_count", DisplayName: "Client Count", Type: "count", SQL: "COUNT(DISTINCT client_id)"},
		{Name: "total_aum", DisplayName: "Total AUM", Type: "sum", SQL: "SUM(client_aum)", Format: "currency"},
		{Name: "avg_aum_per_advisor", DisplayName: "Avg AUM per Advisor", Type: "avg", SQL: "AVG(client_aum)", Format: "currency"},
		{Name: "avg_clients_per_advisor", DisplayName: "Avg Clients per Advisor", Type: "avg", SQL: "AVG(COUNT(DISTINCT client_id))", Format: "number"},
	}
}
