package domain

// retain_unused_cache_symbols.go
// Small retention shim to reference intentionally-kept fields in the cache package
// so staticcheck does not report them as unused.
func init() {
	var r RedisDecisionCache
	_ = r._ttl
}
