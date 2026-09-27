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
const (
	SourceKey = "@source_key"
	AsOfKey   = "@as_of"
)

var identifierKey = regexp.MustCompile(`^id:([A-Z][A-Z0-9_]{1,39})$`)

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
	_, ok := IdentifierType(key)
	return ok
}

// looksLikeMasteringKey catches a malformed one (id:isin, @sourcekey), which
// must be refused rather than looked up as a field.
func looksLikeMasteringKey(key string) bool {
	return strings.HasPrefix(key, "id:") || strings.HasPrefix(key, "@")
}
