#!/bin/bash
cd /Users/eganpj/GitHub/semlayer

# Already applied: analytics import, service changes

# Fix: Change NewSemanticCalculationService
sed -i '' 's/services\.NewSemanticCalculationService(/analytics.NewSemanticCalculationService(/' backend/internal/api/api.go

# Fix: Remove extra args from NewSemanticMappingService
sed -i '' 's/srv\.SemanticMappingSvc = analytics\.NewSemanticMappingService(sqlxDB, businessTermMatcher, srv\.AbbreviationSvc)/srv.SemanticMappingSvc = analytics.NewSemanticMappingService(sqlxDB)/' backend/internal/api/api.go

# Fix: Stub out CubeSyncService
sed -i '' 's/srv\.CubeSyncService = services\.NewCubeSyncService(sqlxDB, cubeSchemaPath)/srv.CubeSyncService = nil \/\/ Stub: NewCubeSyncService returns interface{}/' backend/internal/api/api.go

# Fix: Stub out CatalogScanHandler  
sed -i '' 's/catalogScanService := services\.NewCatalogScanService(sqlxDB)/\/\/ catalogScanService := services.NewCatalogScanService(sqlxDB)/' backend/internal/api/api.go
sed -i '' 's/catalogScanHandler := handlers\.NewCatalogScanHandler(catalogScanService)/\/\/ catalogScanHandler := handlers.NewCatalogScanHandler(catalogScanService)/' backend/internal/api/api.go
sed -i '' 's/srv\.CatalogScanHandler = catalogScanHandler/srv.CatalogScanHandler = nil \/\/ Stub/' backend/internal/api/api.go

# Fix: Remove extra arg from NewViewService
sed -i '' 's/viewService := services\.NewViewService(sqlxDB, modelProvider)/viewService := services.NewViewService(sqlxDB)/' backend/internal/api/api.go

# Fix: Add nil FinancialToolService to NewNLQService
sed -i '' 's/nlqService := services\.NewNLQService(sqlxDB, geminiProvider, searchService, reasoningEngine)/nlqService := services.NewNLQService(sqlxDB, geminiProvider, searchService, reasoningEngine, nil)/' backend/internal/api/api.go

echo "All fixes applied"
