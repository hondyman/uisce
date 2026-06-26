#!/bin/bash

set -e
set -x

# 1. Remove stray module definitions
rm -f backend/models/go.mod backend/models/go.sum
rm -f backend/internal/explorer/security/go.mod backend/internal/explorer/security/go.sum
rm -f backend/internal/crypto/go.mod backend/internal/crypto/go.sum
rm -f backend/internal/cubeengine/go.mod backend/internal/cubeengine/go.sum

# 2. Purge stale entries from go.sum
cd backend
if [ -f "go.sum" ]; then
    mv go.sum go.sum.bak
    grep -v "github.com/eganpj/semlayer/backend/models" go.sum.bak | grep -v "github.com/eganpj/semlayer/backend/internal/explorer/security" | grep -v "github.com/eganpj/semlayer/backend/internal/crypto" | grep -v "github.com/eganpj/semlayer/backend/internal/cubeengine" > go.sum
    rm go.sum.bak
fi

# 3. Clear the module cache
go clean -modcache

# 4. Re-tidy the backend module
go mod tidy
