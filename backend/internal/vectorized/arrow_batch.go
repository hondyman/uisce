package vectorized

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// ColumnVector represents a memory-aligned primitive column buffer
type ColumnVector struct {
	Name    string
	Floats  []float64
	Ints    []int64
	Strings []string
	Dates   []time.Time
}

// ArrowRecordBatch stores columnar vectors without row-allocation overhead
type ArrowRecordBatch struct {
	Columns map[string]*ColumnVector
	NumRows int
}

func NewArrowRecordBatch(numRows int) *ArrowRecordBatch {
	return &ArrowRecordBatch{
		Columns: make(map[string]*ColumnVector),
		NumRows: numRows,
	}
}

// PackedCashflowVector stores binary cashflows for zero-copy deserialization
type PackedCashflowVector struct {
	Dates   []int64   // Unix timestamps (seconds)
	Amounts []float64 // Transaction values
}

func (v *PackedCashflowVector) Serialize() []byte {
	buf := make([]byte, 4+len(v.Dates)*16)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(v.Dates)))
	offset := 4
	for i := 0; i < len(v.Dates); i++ {
		binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(v.Dates[i]))
		bits := math.Float64bits(v.Amounts[i])
		binary.LittleEndian.PutUint64(buf[offset+8:offset+16], bits)
		offset += 16
	}
	return buf
}

func DeserializeCashflowVector(buf []byte) (*PackedCashflowVector, error) {
	if len(buf) < 4 {
		return nil, errors.New("buffer too short")
	}
	count := int(binary.LittleEndian.Uint32(buf[0:4]))
	if len(buf) < 4+count*16 {
		return nil, errors.New("corrupt buffer length")
	}
	dates := make([]int64, count)
	amounts := make([]float64, count)
	offset := 4
	for i := 0; i < count; i++ {
		dates[i] = int64(binary.LittleEndian.Uint64(buf[offset : offset+8]))
		bits := binary.LittleEndian.Uint64(buf[offset+8 : offset+16])
		amounts[i] = math.Float64frombits(bits)
		offset += 16
	}
	return &PackedCashflowVector{Dates: dates, Amounts: amounts}, nil
}
