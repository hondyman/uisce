package wealth

import (
	"context"

	"github.com/hondyman/semlayer/backend/internal/calcengine"
	"github.com/hondyman/semlayer/backend/pkg/meta"
	"github.com/jmoiron/sqlx"
)

// Bootstrap initializes wealth transfer integration with core platform
type Bootstrap struct {
	db              *sqlx.DB
	calcEngine      calcengine.CalcEngine
	metaService     *meta.Service
	hasuraGenerator *meta.HasuraMetadataGenerator
}

// NewBootstrap creates a new bootstrap instance
func NewBootstrap(
	db *sqlx.DB,
	calcEngine calcengine.CalcEngine,
	metaService *meta.Service,
	hasuraGenerator *meta.HasuraMetadataGenerator,
) *Bootstrap {
	return &Bootstrap{
		db:              db,
		calcEngine:      calcEngine,
		metaService:     metaService,
		hasuraGenerator: hasuraGenerator,
	}
}

// InitializeWealthTransfer bootstraps wealth transfer with core platform integration
func (b *Bootstrap) InitializeWealthTransfer(ctx context.Context, tenantID string) error {
	// 1. Register BusinessObjects in metadata system
	if err := RegisterWealthTransferBusinessObjects(ctx, b.metaService, tenantID); err != nil {
		return err
	}

	// 2. Register enums
	if err := RegisterWealthTransferEnums(ctx, b.metaService, tenantID); err != nil {
		return err
	}

	// 3. Register tax calculation metrics in CalcEngine
	if err := RegisterWealthTaxMetrics(ctx, b.calcEngine, tenantID); err != nil {
		return err
	}

	// 4. Generate and apply Hasura metadata
	hasuraIntegration := NewWealthTransferHasuraIntegration(b.hasuraGenerator, b.metaService)
	if err := hasuraIntegration.GenerateAndApplyMetadata(ctx, tenantID); err != nil {
		return err
	}

	return nil
}

// NewIntegratedTaxService creates a tax service using core CalcEngine
func (b *Bootstrap) NewIntegratedTaxService() *TaxCalcEngineAdapter {
	return NewTaxCalcEngineAdapter(b.calcEngine)
}

// NewIntegratedFamilyOfficeService creates family office service with BO integration
func (b *Bootstrap) NewIntegratedFamilyOfficeService() *FamilyOfficeService {
	return NewFamilyOfficeService(nil)
}

// Example usage in main.go:
/*
func main() {
	// ... existing setup ...

	// Create bootstrap
	bootstrap := wealth.NewBootstrap(
		db,
		calcEngine,
		metaService,
		hasuraGenerator,
	)

	// Initialize wealth transfer with core integration
	if err := bootstrap.InitializeWealthTransfer(ctx, tenantID); err != nil {
		log.Fatal(err)
	}

	// Use integrated services
	taxService := bootstrap.NewIntegratedTaxService() // Uses CalcEngine
	familyService := bootstrap.NewIntegratedFamilyOfficeService() // Uses BusinessObjects

	// ... rest of setup ...
}
*/
