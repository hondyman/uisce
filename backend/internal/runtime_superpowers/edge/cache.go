package edge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type CacheParams struct {
	TenantID string
	PageID   string
	Params   map[string]string
}

func GenerateCacheKey(p CacheParams) string {
	// hash(params)
	keys := make([]string, 0, len(p.Params))
	for k := range p.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(p.Params[k])
		sb.WriteString(";")
	}

	hash := sha256.Sum256([]byte(sb.String()))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("edge:%s:%s:%s", p.TenantID, p.PageID, hashStr)
}

func GetCacheDirectives(volatility string) string {
	switch volatility {
	case "static":
		return "public, max-age=31536000, immutable" // 1 year
	case "low":
		return "public, max-age=3600, stale-while-revalidate=600" // 1 hour
	case "high":
		return "public, max-age=60, stale-while-revalidate=10" // 1 minute
	default:
		return "no-cache"
	}
}
