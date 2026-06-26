package catalogsync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// ComputeNodeHash builds a deterministic hash over the semantic fields of a node.
func ComputeNodeHash(n NodeInput) (string, error) {
	payload := map[string]any{
		"typeId":        n.TypeID.String(),
		"name":          n.Name,
		"description":   n.Description,
		"qualifiedPath": n.QualifiedPath,
		"parentId":      uuidStringPtr(n.ParentID),
		"properties":    n.Properties,
		"config":        n.Config,
		"tenantId":      n.TenantID.String(),
		"tenantSource":  uuidStringPtr(n.TenantDatasourceID),
	}
	return stableHash(payload)
}

// ComputeEdgeHash builds a deterministic hash over the semantic fields of an edge.
func ComputeEdgeHash(e EdgeInput) (string, error) {
	payload := map[string]any{
		"source":           e.SourceNodeID.String(),
		"target":           e.TargetNodeID.String(),
		"edgeTypeId":       e.EdgeTypeID.String(),
		"edgeType":         e.EdgeType,
		"relationship":     stringPtr(e.RelationshipType),
		"properties":       e.Properties,
		"tenantId":         e.TenantID.String(),
		"tenantDatasource": uuidStringPtr(e.TenantDatasourceID),
	}
	return stableHash(payload)
}

func stableHash(v any) (string, error) {
	buf := &bytes.Buffer{}
	if err := writeStableJSON(buf, v); err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writeStableJSON(buf *bytes.Buffer, v any) error {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool, string, float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(b)
	case json.RawMessage:
		buf.Write(val)
	case []any:
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeStableJSON(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		buf.WriteByte('{')
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			if err := writeStableJSON(buf, val[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("unsupported type in stable json: %T", v)
	}
	return nil
}

func uuidStringPtr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func stringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
