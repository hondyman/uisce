package vm

import (
	"encoding/json"
	"math"
	"sync"
)

// Bitmask bits for FastRecord.Present[i].
const (
	HasNum  uint8 = 1 << 0
	HasBool uint8 = 1 << 1
	HasStr  uint8 = 1 << 2
	HasEnum uint8 = 1 << 3
	HasFNum uint8 = 1 << 4
)

// EnumMissing is the zero-value for an unassigned enum ID.
const EnumMissing uint32 = 0

// FastRecord is the dense, projection-ready view of an MDM record.
// All slices are sized to syms.Num() at construction time.
// All field access is O(1) array indexing by SymbolID.
type FastRecord struct {
	NumVals  []int64
	BoolVals []bool
	StrVals  []string
	EnumVals []uint32
	FNumVals []float64
	Present  []uint8
}

var fastRecordPool = sync.Pool{
	New: func() any { return &FastRecord{} },
}

// GetFastRecord retrieves a FastRecord from the pool, sizing it to the dictionary.
// WARNING: Caller MUST call PutFastRecord(r) when done to prevent pool leaks.
func GetFastRecord(syms *SymbolDict) *FastRecord {
	r := fastRecordPool.Get().(*FastRecord)
	cap := int(syms.Num())

	if cap == 0 {
		return r
	}

	if cap > len(r.NumVals) {
		r.NumVals = make([]int64, cap)
		r.BoolVals = make([]bool, cap)
		r.StrVals = make([]string, cap)
		r.EnumVals = make([]uint32, cap)
		r.FNumVals = make([]float64, cap)
		r.Present = make([]uint8, cap)
	} else {
		// Go 1.21+ builtin: zero out slices efficiently without allocation
		clear(r.NumVals[:cap])
		clear(r.BoolVals[:cap])
		clear(r.StrVals[:cap])
		clear(r.EnumVals[:cap])
		clear(r.FNumVals[:cap])
		clear(r.Present[:cap])
	}

	return r
}

// PutFastRecord returns the record to the pool.
func PutFastRecord(r *FastRecord) {
	fastRecordPool.Put(r)
}

// Project flattens a map[string]any MDM record into a FastRecord
// using the supplied dictionaries. Caller MUST call PutFastRecord(rec).
func Project(record map[string]any, syms *SymbolDict, enums *EnumDict) *FastRecord {
	r := GetFastRecord(syms)
	if len(r.NumVals) == 0 {
		return r
	}
	projectRecursive("", record, r, syms, enums)
	return r
}

func projectRecursive(prefix string, data any, r *FastRecord, syms *SymbolDict, enums *EnumDict) {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			var newPrefix string
			if prefix == "" {
				newPrefix = k
			} else {
				newPrefix = prefix + "." + k
			}
			projectRecursive(newPrefix, val, r, syms, enums)
		}

	case float64:
		if id, ok := syms.Resolve(prefix); ok {
			r.FNumVals[id] = v
			r.Present[id] |= HasFNum
			if v == float64(int64(v)) && !math.IsInf(v, 0) {
				r.NumVals[id] = int64(v)
				r.Present[id] |= HasNum
			}
		}

	case int:
		if id, ok := syms.Resolve(prefix); ok {
			r.NumVals[id] = int64(v)
			r.Present[id] |= HasNum
			r.FNumVals[id] = float64(v)
			r.Present[id] |= HasFNum
		}

	case int64:
		if id, ok := syms.Resolve(prefix); ok {
			r.NumVals[id] = v
			r.Present[id] |= HasNum
			r.FNumVals[id] = float64(v)
			r.Present[id] |= HasFNum
		}

	case bool:
		if id, ok := syms.Resolve(prefix); ok {
			r.BoolVals[id] = v
			r.Present[id] |= HasBool
		}

	case string:
		if id, ok := syms.Resolve(prefix); ok {
			r.StrVals[id] = v
			r.Present[id] |= HasStr

			// If this string matches an interned enum, cache the enum ID as well
			if eid, ok := enums.ID(v); ok {
				r.EnumVals[id] = eid
				r.Present[id] |= HasEnum
			}
		}

	case json.Number:
		if id, ok := syms.Resolve(prefix); ok {
			if n, err := v.Float64(); err == nil {
				r.FNumVals[id] = n
				r.Present[id] |= HasFNum
			}
		}
	}
}