package cold

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
)

// MerkleProofStep represents a single node sibling on the authentication path to the root
type MerkleProofStep struct {
	SiblingHash string `json:"sibling_hash"`
	IsRight     bool   `json:"is_right"`
}

// MerkleTree represents a binary Merkle tree over canonical compliance evaluation leaves
type MerkleTree struct {
	Leaves [][]byte
	Levels [][][]byte
	Root   string
}

// CanonicalEvaluationLeaf computes a deterministic SHA-256 leaf hash for an evaluation record
func CanonicalEvaluationLeaf(lineageID, tenantID, ruleID string, ruleVersion int, actionTaken string, ingestLSN int64, inputParams, metricSnapshots json.RawMessage) ([]byte, error) {
	// Canonicalize input_params and metric_snapshots using RFC 8785 JCS
	canonInput, err := canonical.Transform(inputParams)
	if err != nil {
		canonInput = []byte("{}")
	}
	canonMetrics, err := canonical.Transform(metricSnapshots)
	if err != nil {
		canonMetrics = []byte("{}")
	}

	h := sha256.New()
	h.Write([]byte(lineageID))
	h.Write([]byte(":"))
	h.Write([]byte(tenantID))
	h.Write([]byte(":"))
	h.Write([]byte(ruleID))
	h.Write([]byte(":"))
	h.Write([]byte(strconv.Itoa(ruleVersion)))
	h.Write([]byte(":"))
	h.Write([]byte(actionTaken))
	h.Write([]byte(":"))
	h.Write([]byte(strconv.FormatInt(ingestLSN, 10)))
	h.Write([]byte(":"))
	h.Write(canonInput)
	h.Write([]byte(":"))
	h.Write(canonMetrics)

	sum := h.Sum(nil)
	return sum, nil
}

// BuildMerkleTree constructs a binary Merkle tree from a sequence of leaf hashes
func BuildMerkleTree(leaves [][]byte) (*MerkleTree, string) {
	if len(leaves) == 0 {
		emptyRoot := sha256.Sum256([]byte{})
		rootHex := hex.EncodeToString(emptyRoot[:])
		return &MerkleTree{
			Leaves: nil,
			Levels: [][][]byte{{emptyRoot[:]}},
			Root:   rootHex,
		}, rootHex
	}

	// Make a defensive copy
	currentLevel := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		leafCopy := make([]byte, len(leaf))
		copy(leafCopy, leaf)
		currentLevel[i] = leafCopy
	}

	var levels [][][]byte
	levels = append(levels, currentLevel)

	for len(currentLevel) > 1 {
		var nextLevel [][]byte
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined := hashPair(currentLevel[i], currentLevel[i+1])
				nextLevel = append(nextLevel, combined)
			} else {
				// Odd node is promoted / hashed with itself for balanced binary tree
				combined := hashPair(currentLevel[i], currentLevel[i])
				nextLevel = append(nextLevel, combined)
			}
		}
		levels = append(levels, nextLevel)
		currentLevel = nextLevel
	}

	rootHex := hex.EncodeToString(currentLevel[0])
	return &MerkleTree{
		Leaves: leaves,
		Levels: levels,
		Root:   rootHex,
	}, rootHex
}

func hashPair(left, right []byte) []byte {
	h := sha256.New()
	// Prefix with 0x01 domain separator for internal nodes (RFC 6962 standard)
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// GenerateProof produces the inclusion proof path for a leaf at index
func (t *MerkleTree) GenerateProof(leafIndex int) ([]MerkleProofStep, error) {
	if leafIndex < 0 || leafIndex >= len(t.Leaves) {
		return nil, fmt.Errorf("leaf index %d out of bounds (total leaves: %d)", leafIndex, len(t.Leaves))
	}

	var proof []MerkleProofStep
	currentIndex := leafIndex

	for levelIdx := 0; levelIdx < len(t.Levels)-1; levelIdx++ {
		currentLevel := t.Levels[levelIdx]
		var siblingHash []byte
		var isRight bool

		if currentIndex%2 == 0 {
			// Even index: sibling is on the right
			if currentIndex+1 < len(currentLevel) {
				siblingHash = currentLevel[currentIndex+1]
			} else {
				siblingHash = currentLevel[currentIndex] // duplicate odd node
			}
			isRight = true
		} else {
			// Odd index: sibling is on the left
			siblingHash = currentLevel[currentIndex-1]
			isRight = false
		}

		proof = append(proof, MerkleProofStep{
			SiblingHash: hex.EncodeToString(siblingHash),
			IsRight:     isRight,
		})

		currentIndex /= 2
	}

	return proof, nil
}

// VerifyInclusionProof verifies that a leaf hash belongs to the Merkle tree with rootHex using proof
func VerifyInclusionProof(leafHash []byte, proof []MerkleProofStep, rootHex string) bool {
	current := make([]byte, len(leafHash))
	copy(current, leafHash)

	for _, step := range proof {
		siblingBytes, err := hex.DecodeString(step.SiblingHash)
		if err != nil {
			return false
		}

		if step.IsRight {
			current = hashPair(current, siblingBytes)
		} else {
			current = hashPair(siblingBytes, current)
		}
	}

	computedRoot := hex.EncodeToString(current)
	return computedRoot == rootHex
}

// SortEvaluationRecords deterministically sorts records by (IngestLSN ASC, LineageID ASC)
func SortEvaluationRecords(records []CanonicalRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].IngestLSN != records[j].IngestLSN {
			return records[i].IngestLSN < records[j].IngestLSN
		}
		return records[i].LineageID < records[j].LineageID
	})
}

// CanonicalRecord is the standard structure for cold tier Parquet and Merkle processing
type CanonicalRecord struct {
	LineageID       string `parquet:"lineage_id"`
	EvaluatedAt     string `parquet:"evaluated_at"` // RFC3339 UTC fixed precision
	TenantID        string `parquet:"tenant_id"`
	OrderID         string `parquet:"order_id"`
	RuleID          string `parquet:"rule_id"`
	RuleVersion     int32  `parquet:"rule_version"`
	ActionTaken     string `parquet:"action_taken"`
	Passed          bool   `parquet:"passed"`
	LatencyMicros   int64  `parquet:"latency_micros"`
	EvaluationHash  string `parquet:"evaluation_hash"`
	IngestLSN       int64  `parquet:"ingest_lsn"`
	InputParams     string `parquet:"input_params"`     // Canonical RFC 8785 JCS string
	MetricSnapshots string `parquet:"metric_snapshots"` // Canonical RFC 8785 JCS string
	CreatedAt       string `parquet:"created_at"`
}
