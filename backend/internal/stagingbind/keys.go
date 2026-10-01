package stagingbind

import (
	"regexp"
	"strings"
)

// Besides BO fields, a binding carries the keys mastering needs from a
// source: its identifiers and its own record key. These are not BO fields
// (an entity has many identifiers, kept in its identifier table), so they
// are named, not looked up in the catalog:
//
//	id:<TYPE>    a source column holding an identifier of that type (id:ISIN)
//	@source_key  the source's own record key, the stable handle for the xref
//	@as_of       the source's as-of time, for most-recent and staleness rules
//	value:<TYPE> a source column holding a time-series value of that type -
//	             a price type for the price master (value:OFFICIAL_CLOSE), so
//	             one vendor row with several price columns is several prices
const (
	SourceKey = "@source_key"
	AsOfKey   = "@as_of"
)

var (
	identifierKey = regexp.MustCompile(`^id:([A-Z][A-Z0-9_]{1,39})$`)
	valueKey      = regexp.MustCompile(`^value:([A-Z][A-Z0-9_]{1,39})$`)
)

// ValueType returns the series type of a value:<TYPE> key.
func ValueType(key string) (string, bool) {
	m := valueKey.FindStringSubmatch(key)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// IdentifierType returns the identifier type of an id:<TYPE> key.
func IdentifierType(key string) (string, bool) {
	m := identifierKey.FindStringSubmatch(key)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// IsMasteringKey reports a well-formed key that is not a BO field.
func IsMasteringKey(key string) bool {
	if key == SourceKey || key == AsOfKey {
		return true
	}
	if _, ok := IdentifierType(key); ok {
		return true
	}
	_, ok := ValueType(key)
	return ok
}

// looksLikeMasteringKey catches a malformed one (id:isin, @sourcekey), which
// must be refused rather than looked up as a field.
func looksLikeMasteringKey(key string) bool {
	return strings.HasPrefix(key, "id:") || strings.HasPrefix(key, "value:") || strings.HasPrefix(key, "@")
}
