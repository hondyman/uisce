package tiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
)

// TagMapping is the per-(tenant, fix_version, msg_type, fix_tag) row
// from fix_tenant_tag_mapping. Used by both FixTagMap (forward direction:
// raw tags → semantic fields) and FixOrderEmit (reverse: semantic
// fields → raw tags).
type TagMapping struct {
	FixTag        int
	SemanticField string
	Required      bool
	DefaultValue  string
	TransformFn   string // "upper", "numeric", "parse_iso_currency", etc.
	TenantOwned   bool   // true when this row is the tenant's own (not gold-copy)
}

// TagMappingLoader is the dependency FixTagMap needs to fetch the
// per-tenant tag-mapping rows. The implementation is in the data-
// pipeline tile adapter (reads fix_tenant_tag_mapping with the GSIFI
// OR-clause). Tests use the in-memory FakeTagMappingLoader.
type TagMappingLoader interface {
	Load(ctx context.Context, tenantID, brokerID, fixVersion, msgType string) ([]TagMapping, error)
}

// FakeTagMappingLoader is a tiny in-memory loader for tests.
type FakeTagMappingLoader struct {
	Mappings map[string][]TagMapping // keyed by "<fixVersion>|<msgType>"
}

// Load implements TagMappingLoader.
func (f *FakeTagMappingLoader) Load(_ context.Context, _, _, fixVersion, msgType string) ([]TagMapping, error) {
	key := fixVersion + "|" + msgType
	return f.Mappings[key], nil
}

// FixTagMap applies per-tenant tag → semantic-field mapping to each
// record's parsed tags. Per HANDOFF_FIX_OVER_PIPELINE.md §9 `fix_tag_map`.
//
// Precedence rule (HANDOFF §9): tenant-owned rows win over gold-copy
// rows when both exist for the same (fix_version, msg_type, fix_tag).
// The loader must return tenant-owned rows last in the slice; this
// function picks the LAST entry per fix_tag so the tenant override
// survives.
func FixTagMap(loader TagMappingLoader) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		t := TenantFromContext(ctx)
		if t.TenantID == "" {
			return records, []string{"fix_tag_map: missing tenant context"}, nil
		}

		out := make([]Record, 0, len(records))
		var errs []string

		for _, rec := range records {
			msgType, _ := rec["msg_type"].(string)
			fixVersion := "FIX.4.4" // default per fix_tenant_config.fix_version

			mappings, err := loader.Load(ctx, t.TenantID, t.BrokerID, fixVersion, msgType)
			if err != nil {
				errs = append(errs, fmt.Sprintf("fix_tag_map: load mappings: %v", err))
				out = append(out, rec)
				continue
			}

			// Group by fix_tag; later entries win.
			merged := make(map[int]TagMapping, len(mappings))
			for _, m := range mappings {
				merged[m.FixTag] = m
			}

			tagsAny, _ := rec["tags"].(map[string]string)
			semantic := make(map[string]any, len(merged))

			for tagNum, m := range merged {
				raw, ok := tagsAny[strconv.Itoa(tagNum)]
				if !ok {
					if m.Required && m.DefaultValue != "" {
						raw = m.DefaultValue
					} else if m.Required {
						errs = append(errs, fmt.Sprintf("fix_tag_map: required tag %d missing", tagNum))
						continue
					} else {
						continue
					}
				}
				semantic[m.SemanticField] = applyTransform(raw, m.TransformFn)
			}

			newRec := make(Record, len(rec)+1)
			for k, v := range rec {
				newRec[k] = v
			}
			newRec["semantic"] = semantic
			out = append(out, newRec)
		}

		return out, errs, nil
	}
}

// FixOrderEmit is the reverse direction: takes semantic fields and
// emits a tag/value map suitable for an outbound FIX message. Used by
// FIXOrderEntryWorkflow via the `fix_order_emit` tile (HANDOFF §9).
//
// Idempotency dedup key (HANDOFF §19 outbound): (tenant_id, ClOrdID,
// broker_id). When ClOrdID is missing, falls back to a SHA-256 of the
// record itself (first 16 hex chars).
func FixOrderEmit(loader TagMappingLoader, msgType string) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		t := TenantFromContext(ctx)
		if t.TenantID == "" {
			return records, []string{"fix_order_emit: missing tenant context"}, nil
		}

		out := make([]Record, 0, len(records))
		var errs []string

		mappings, err := loader.Load(ctx, t.TenantID, t.BrokerID, "FIX.4.4", msgType)
		if err != nil {
			return records, []string{fmt.Sprintf("fix_order_emit: load mappings: %v", err)}, nil
		}

		// Reverse index: semantic_field → tag
		byField := make(map[string]TagMapping, len(mappings))
		for _, m := range mappings {
			byField[m.SemanticField] = m
		}

		for _, rec := range records {
			semanticAny, ok := rec["semantic"].(map[string]any)
			if !ok {
				errs = append(errs, "fix_order_emit: missing semantic field")
				continue
			}

			tagsOut := make(map[string]string, len(semanticAny))
			for field, val := range semanticAny {
				m, ok := byField[field]
				if !ok {
					continue
				}
				tagsOut[strconv.Itoa(m.FixTag)] = fmt.Sprintf("%v", val)
			}

			// Idempotency dedup key (outbound): prefer ClOrdID, fall
			// back to SHA-256(record) prefix.
			clOrdID := tagsOut[strconv.Itoa(11)] // tag 11 = ClOrdID
			if clOrdID == "" {
				clOrdID = recordHash(rec)[:16]
			}

			newRec := make(Record, len(rec)+2)
			for k, v := range rec {
				newRec[k] = v
			}
			newRec["tags"] = tagsOut
			newRec["cl_ord_id"] = clOrdID
			newRec["direction"] = "outbound"
			out = append(out, newRec)
		}

		return out, errs, nil
	}
}

// applyTransform is a tiny built-in set of type coercion helpers.
// Extend as new broker requirements emerge.
func applyTransform(raw, fn string) any {
	switch fn {
	case "upper":
		return upper(raw)
	case "lower":
		return lower(raw)
	case "numeric":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return raw
		}
		return f
	case "int":
		i, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return raw
		}
		return i
	case "":
		return raw
	default:
		// Unknown transform_fn: leave raw.
		return raw
	}
}

func upper(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		out[i] = c
	}
	return string(out)
}

func lower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}

// recordHash returns a SHA-256 hex digest of the record's semantic
// field set (deterministic for idempotency fallback). Keys are sorted
// so map iteration order doesn't change the hash.
func recordHash(rec Record) string {
	h := sha256.New()
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(fmt.Sprintf("%v", rec[k])))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
