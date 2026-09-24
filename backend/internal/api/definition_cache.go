package api

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Version constants — bump NamingLogicVersion when naming rules change,
// bump AbbreviationsVersion when the abbreviation dictionary changes.
// Two columns that derive the same name with different dictionary versions
// produce different cache keys, so stale definitions are never reused.
const (
	NamingLogicVersion  = "v1"
	AbbreviationsVersion = "v1"
)

// buildDefinitionCacheKey computes the versioned cache key for a definition.
//
// Key = sha256(NamingLogicVersion || AbbreviationsVersion || derivedTokensJoined || contextPart)
//
// contextPart is tableSchemaContext only when the derivation is ContextSensitive
// (i.e., the name depended on table context). For non-context-sensitive names,
// contextPart is empty — same column name in 3 tables shares one definition.
func buildDefinitionCacheKey(derivedTokens []string, contextPart string, contextSensitive bool) string {
	tokensJoined := strings.Join(derivedTokens, "|")
	cp := ""
	if contextSensitive {
		cp = contextPart
	}
	input := fmt.Sprintf("%s||%s||%s||%s", NamingLogicVersion, AbbreviationsVersion, tokensJoined, cp)
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h)
}
