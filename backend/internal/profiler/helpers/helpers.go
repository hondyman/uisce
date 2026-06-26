package helpers

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

// ColumnProfile represents the profile data for a database column
type ColumnProfile struct {
	DataSource       string
	Schema           string
	TableName        string
	ColumnName       string
	DataType         string
	Cardinality      int64
	MinLength        int
	MaxLength        int
	AvgLength        float64
	MinValue         float64
	MaxValue         float64
	AvgValue         float64
	StdDev           float64
	FrequentValues   []string
	InferredPatterns []string
	BloomFilter      []byte
	CreatedAt        time.Time
}

// FormatValueForBloom normalizes a value to a string suitable for bloom filter keys
func FormatValueForBloom(v interface{}) string {
	switch vv := v.(type) {
	case string:
		return vv
	case []byte:
		return string(vv)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", vv)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ComputeProfile analyzes a sample of values and computes profile statistics
func ComputeProfile(values []interface{}) *ColumnProfile {
	var strVals []string
	var numVals []float64
	var lengths []int
	var nums []float64

	for _, v := range values {
		switch vv := v.(type) {
		case string:
			strVals = append(strVals, vv)
			lengths = append(lengths, len(vv))
			if n, err := strconv.ParseFloat(vv, 64); err == nil {
				nums = append(nums, n)
			}
		case int64:
			numVals = append(numVals, float64(vv))
		case float64:
			numVals = append(numVals, vv)
		}
	}

	prof := &ColumnProfile{}

	if len(numVals) > len(strVals)/2 {
		prof.DataType = "numeric"
	} else {
		prof.DataType = "text"
	}

	unique := make(map[string]struct{})
	for _, v := range values {
		unique[FormatValueForBloom(v)] = struct{}{}
	}
	prof.Cardinality = int64(len(unique))

	if len(lengths) > 0 {
		sort.Ints(lengths)
		prof.MinLength = lengths[0]
		prof.MaxLength = lengths[len(lengths)-1]
		sum := 0
		for _, l := range lengths {
			sum += l
		}
		prof.AvgLength = float64(sum) / float64(len(lengths))
	}

	if len(nums) > 0 {
		sort.Float64s(nums)
		prof.MinValue = nums[0]
		prof.MaxValue = nums[len(nums)-1]
		sum := 0.0
		for _, n := range nums {
			sum += n
		}
		prof.AvgValue = sum / float64(len(nums))

		variance := 0.0
		for _, n := range nums {
			variance += math.Pow(n-prof.AvgValue, 2)
		}
		prof.StdDev = math.Sqrt(variance / float64(len(nums)))
	}

	freq := make(map[string]int)
	for _, v := range values {
		key := FormatValueForBloom(v)
		freq[key]++
	}
	type kv struct {
		Key string
		Val int
	}
	var ss []kv
	for k, v := range freq {
		ss = append(ss, kv{Key: k, Val: v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Val > ss[j].Val })
	{
		limit := min(10, len(ss))
		prof.FrequentValues = make([]string, limit)
		for i := 0; i < limit; i++ {
			prof.FrequentValues[i] = ss[i].Key
		}
	}

	prof.InferredPatterns = InferPatterns(strVals)

	return prof
}

// CreateBloomFilter creates a bloom filter from values with the specified false positive rate
func CreateBloomFilter(values []interface{}, fpRate float64) ([]byte, error) {
	bf := bloom.NewWithEstimates(uint(len(values)), fpRate)
	for _, v := range values {
		key := FormatValueForBloom(v)
		bf.AddString(key)
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(bf); err != nil {
		return nil, fmt.Errorf("failed to encode bloom filter: %w", err)
	}
	return buf.Bytes(), nil
}

// InferPatterns analyzes string values to infer common patterns like email, phone, etc.
func InferPatterns(strVals []string) []string {
	patterns := make(map[string]struct{})
	emailRe := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRe := regexp.MustCompile(`^\+?[\d\s\-\(\)]{10,15}$`)
	for _, s := range strVals {
		if emailRe.MatchString(s) {
			patterns["email"] = struct{}{}
		}
		if phoneRe.MatchString(s) {
			patterns["phone"] = struct{}{}
		}
		if len(patterns) == 2 {
			break
		}
	}
	var inferred []string
	for p := range patterns {
		inferred = append(inferred, p)
	}
	return inferred
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
