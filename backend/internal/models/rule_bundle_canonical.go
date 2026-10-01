package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// CanonicalBundleBytes produces a deterministic byte serialization of the bundle's
// rule set. It is the sole input to Checksum computation:
//
//   - rules are processed sorted by RuleKey
//   - each field is length-prefixed so concatenation is unambiguous
//   - RuleAST is canonicalized via vm.Compact (sorted keys, shortest floats)
//   - tombstoned rules (Deleted=true) contribute an empty AST slot, so
//     deletions change the checksum
//
// Bundle metadata (CreatedAt, ExportedFrom, Checksum itself) is deliberately
// excluded — re-exporting the same rules from different environments must
// yield the same checksum.
func CanonicalBundleBytes(b *RuleBundle) ([]byte, error) {
	sorted := make([]PortableRuleSpec, len(b.Rules))
	copy(sorted, b.Rules)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RuleKey < sorted[j].RuleKey })

	var buf []byte
	appendField := func(field string) {
		buf = append(buf, byte(len(field)>>24), byte(len(field)>>16), byte(len(field)>>8), byte(len(field)))
		buf = append(buf, field...)
	}

	for _, r := range sorted {
		astCanonical := ""
		if !r.Deleted && len(r.RuleAST) > 0 {
			var node vm.RuleNode
			if err := json.Unmarshal(r.RuleAST, &node); err != nil {
				return nil, fmt.Errorf("canonicalize rule %q: parse AST: %w", r.RuleKey, err)
			}
			compact, err := vm.Compact(node)
			if err != nil {
				return nil, fmt.Errorf("canonicalize rule %q: %w", r.RuleKey, err)
			}
			astCanonical = string(compact)
		}
		appendField(r.RuleKey)
		appendField(astCanonical)
		appendField(r.Severity)
		appendField(r.Timing)
		appendField(r.Domain)
		appendField(r.Category)
		if r.Deleted {
			appendField("deleted")
		} else {
			appendField("live")
		}
	}
	return buf, nil
}

// ComputeChecksum populates b.Checksum from the canonical rule set.
func (b *RuleBundle) ComputeChecksum() error {
	canonical, err := CanonicalBundleBytes(b)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	b.Checksum = hex.EncodeToString(sum[:])
	return nil
}

// VerifyChecksum recomputes and compares against the stored checksum.
func (b *RuleBundle) VerifyChecksum() error {
	stored := b.Checksum
	if err := b.ComputeChecksum(); err != nil {
		return err
	}
	if stored != b.Checksum {
		return fmt.Errorf("bundle checksum mismatch: manifest=%s computed=%s", stored, b.Checksum)
	}
	return nil
}
