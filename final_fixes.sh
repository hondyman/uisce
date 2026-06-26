#!/bin/bash
cd /Users/eganpj/GitHub/semlayer

# Fix NLQService - find the actual line and fix it
sed -i '' 's/services\.NewNLQService(sqlxDB, geminiProvider, searchService, reasoningEngine)/services.NewNLQService(sqlxDB, geminiProvider, searchService, reasoningEngine, nil)/' backend/internal/api/api.go

# Fix SearchRequest conversions for SemanticTerms
perl -i -pe 's/(terms, err := srv\.SemanticMappingSvc\.SearchSemanticTerms\(r\.Context\(\), )req(, tenantID, tenantDatasourceID\))/\1analytics.SearchRequest{Query: req.Query, Limit: req.Limit}\2/' backend/internal/api/api.go

# Fix SearchRequest conversions for BusinessTerms
perl -i -pe 's/(terms, err := srv\.SemanticMappingSvc\.SearchBusinessTerms\(r\.Context\(\), )req(, tenantID, tenantDatasourceID\))/\1analytics.SearchRequest{Query: req.Query, Limit: req.Limit}\2/' backend/internal/api/api.go

# Fix AbbreviationMap - change from type to value
sed -i '' 's/"abbreviations": services\.AbbreviationMap/"abbreviations": []services.AbbreviationMap{}/' backend/internal/api/api.go

# Fix modelProvider.GetActiveCatalog calls - remove extra args
sed -i '' 's/modelProvider\.GetActiveCatalog(r\.Context(), tenantID, datasourceID)/modelProvider.GetActiveCatalog(datasourceID)/' backend/internal/api/api.go
sed -i '' 's/modelProvider\.GetActiveCatalog(r\.Context(), tenantID, tenantDatasourceID)/modelProvider.GetActiveCatalog(tenantDatasourceID)/' backend/internal/api/api.go

echo "Final fixes applied"
