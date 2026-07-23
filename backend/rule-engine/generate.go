package ruleengine

//go:generate go run ./cmd/generate-types
//go:generate go run ./cmd/generate-schema
//go:generate go run ./cmd/generate-monaco
//go:generate go run ./cmd/generate-version

// This file exists solely to hold the go:generate directive
// that regenerates TypeScript definitions from Go structs.
//
// Run: go generate ./...
// This will execute: go run ./cmd/generate-types
