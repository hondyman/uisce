package wealth

import (
	"context"

	"github.com/hondyman/uisce/backend/internal/calcengine"
	"github.com/hondyman/uisce/backend/pkg/meta"
	"github.com/jmoiron/sqlx"
)

type Bootstrap struct {
	db          *sqlx.DB
	calcEngine  calcengine.CalcEngine
	metaService *meta.Service
}

func NewBootstrap(
	db *sqlx.DB,
	calcEngine calcengine.CalcEngine,
	metaService *meta.Service,
) *Bootstrap {
	return &Bootstrap{
		db:          db,
		calcEngine:  calcEngine,
		metaService: metaService,
	}
}

func (b *Bootstrap) InitializeWealthTransfer(ctx context.Context, tenantID string) error {
	if err := RegisterWealthTransferBusinessObjects(ctx, b.metaService, tenantID); err != nil {
		return err
	}

	if err := RegisterWealthTransferEnums(ctx, b.metaService, tenantID); err != nil {
		return err
	}

	if err := RegisterWealthTaxMetrics(ctx, b.calcEngine, tenantID); err != nil {
		return err
	}

	return nil
}

func (b *Bootstrap) NewIntegratedTaxService() *TaxCalcEngineAdapter {
	return NewTaxCalcEngineAdapter(b.calcEngine)
}

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
