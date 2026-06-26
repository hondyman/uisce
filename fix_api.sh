#!/bin/bash
# Comprehensive fix for api.go to achieve successful compilation

cd /Users/eganpj/GitHub/semlayer

# Fix 1: Update audit.NewService() - remove db parameter
sed -i '' 's/auditSvc := audit\.NewService(db)/auditSvc := audit.NewService()/' backend/internal/api/api.go

# Fix 2: Update SecurityManager signature
sed -i '' 's/secMgr := services\.NewSecurityManager(jwtSecret)/secMgr := services.NewSecurityManager(nil, nil, jwtSecret)/' backend/internal/api/api.go

# Fix 3: Update NewAbbreviationService signature  
sed -i '' 's/srv\.AbbreviationSvc = services\.NewAbbreviationService(db, logging\.GetLogger())/srv.AbbreviationSvc = services.NewAbbreviationService(db)/' backend/internal/api/api.go

# Fix 4: Change to analytics services
sed -i '' 's/srv\.SemanticSvc = services\.NewSemanticService(/srv.SemanticSvc = analytics.NewSemanticService(/' backend/internal/api/api.go
sed -i '' 's/srv\.SemanticMappingSvc = services\.NewSemanticMappingServiceWithAbbreviations(/srv.SemanticMappingSvc = analytics.NewSemanticMappingService(/' backend/internal/api/api.go
sed -i '' 's/srv\.SemanticCalculationSvc = services\.NewSemanticCalculationService(/srv.SemanticCalculationSvc = analytics.NewSemanticCalculationService(/' backend/internal/api/api.go
sed -i '' 's/semanticSvc := services\.NewSemanticModelService(/semanticSvc := analytics.NewSemanticModelService(/' backend/internal/api/api.go

# Fix 5: Update struct field definitions in Server struct
sed -i '' 's/SemanticSvc[[:space:]]*\*services\.SemanticService/SemanticSvc *analytics.SemanticService/' backend/internal/api/api.go
sed -i '' 's/SemanticMappingSvc[[:space:]]*\*services\.SemanticMappingService/SemanticMappingSvc *analytics.SemanticMappingService/' backend/internal/api/api.go  
sed -i '' 's/SemanticCalculationSvc[[:space:]]*\*services\.SemanticCalculationService/SemanticCalculationSvc *analytics.SemanticCalculationService/' backend/internal/api/api.go
sed -i '' 's/CubeSyncService[[:space:]]*\*services\.CubeSyncService/CubeSyncService *analytics.CubeSyncService/' backend/internal/api/api.go

echo "Fixes applied successfully"
