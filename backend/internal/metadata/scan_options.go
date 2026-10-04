package metadata

import (
	"context"
	"encoding/json"
)

// Profiling a scan's data (row counts, unique counts, sample values) reads every column of every table. It is what feeds
// the catalog's data-quality hints, and it is by far the slowest step against a large source. It is optional.
//
// Whether a scan profiles is decided, most specific first:
//
//  1. the request: ScanOptions.ProfileData, carried in the context (the API's "profile_data" field or query parameter);
//  2. the datasource: "profile_data": false in its connection config;
//  3. the default, which is unchanged: profile.
//
// A scan that does not profile records exactly the same structure (columns, keys, constraints, indexes, partitioning,
// routines, triggers), so it is the right scan for a gold-copy template, where only structure is deployed.

// ScanOptions are per-request choices about how a scan runs.
type ScanOptions struct {
	// ProfileData, when set, overrides the datasource's own setting for this scan. Nil means "no opinion".
	ProfileData *bool
}

type scanOptionsKey struct{}

// WithScanOptions returns a context that carries the options for any scan started with it.
func WithScanOptions(ctx context.Context, o ScanOptions) context.Context {
	return context.WithValue(ctx, scanOptionsKey{}, o)
}

// ScanOptionsFrom is the options a context carries; the zero value means none were given.
func ScanOptionsFrom(ctx context.Context) ScanOptions {
	o, _ := ctx.Value(scanOptionsKey{}).(ScanOptions)
	return o
}

// profileData decides whether this scan profiles the data of datasource ds.
func profileData(ctx context.Context, ds DatasourceConfig) bool {
	if o := ScanOptionsFrom(ctx); o.ProfileData != nil {
		return *o.ProfileData
	}
	var cfg struct {
		ProfileData *bool `json:"profile_data"`
	}
	if err := json.Unmarshal([]byte(ds.ConnectionDetails), &cfg); err == nil && cfg.ProfileData != nil {
		return *cfg.ProfileData
	}
	return true
}
