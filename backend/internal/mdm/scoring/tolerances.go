package scoring

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// EvaluateTolerance checks if candidate and golden match according to the specified rule.
func EvaluateTolerance(tol AttributeTolerance, candidateVal, goldenVal string) bool {
	cleanCand := strings.TrimSpace(candidateVal)
	cleanGold := strings.TrimSpace(goldenVal)

	if cleanCand == "" || cleanGold == "" {
		return false
	}

	if cleanCand == cleanGold {
		return true
	}

	switch tol.MatchType {
	case MatchExact:
		return strings.EqualFold(cleanCand, cleanGold)

	case MatchFuzzyJaro:
		threshold := tol.ToleranceVal
		if threshold <= 0 {
			threshold = 0.92
		}
		return JaroWinklerSimilarity(cleanCand, cleanGold) >= threshold

	case MatchNumericBP:
		cNum, err1 := strconv.ParseFloat(cleanCand, 64)
		gNum, err2 := strconv.ParseFloat(cleanGold, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		tolBp := tol.ToleranceVal
		if tolBp <= 0 {
			tolBp = 0.0001 // 1 basis point default
		}
		return math.Abs(cNum-gNum) <= tolBp+1e-9

	case MatchNumericPct:
		cNum, err1 := strconv.ParseFloat(cleanCand, 64)
		gNum, err2 := strconv.ParseFloat(cleanGold, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		if gNum == 0 {
			return cNum == 0
		}
		tolPct := tol.ToleranceVal
		if tolPct <= 0 {
			tolPct = 0.01 // 0.01%
		}
		deltaPct := (math.Abs(cNum-gNum) / math.Abs(gNum)) * 100.0
		return deltaPct <= tolPct

	case MatchDateLag:
		layouts := []string{"2006-01-02", "2006/01/02", "20060102", time.RFC3339}
		var tCand, tGold time.Time
		var errCand, errGold error

		for _, layout := range layouts {
			if tCand.IsZero() {
				tCand, errCand = time.Parse(layout, cleanCand)
			}
			if tGold.IsZero() {
				tGold, errGold = time.Parse(layout, cleanGold)
			}
		}

		if errCand != nil || errGold != nil {
			return false
		}

		maxDays := int(tol.ToleranceVal)
		if maxDays <= 0 {
			maxDays = 1 // default ±1 business day
		}

		diffHours := math.Abs(tCand.Sub(tGold).Hours())
		diffDays := int(math.Ceil(diffHours / 24.0))
		return diffDays <= maxDays

	default:
		return strings.EqualFold(cleanCand, cleanGold)
	}
}

// JaroWinklerSimilarity calculates the Jaro-Winkler metric between two strings (0.0 to 1.0).
func JaroWinklerSimilarity(s1, s2 string) float64 {
	jaro := jaroSimilarity(s1, s2)
	if jaro < 0.7 {
		return jaro
	}

	// Prefix scale constant p = 0.1
	// Maximum 4 common prefix characters
	prefixLen := 0
	maxPrefix := 4
	if len(s1) < maxPrefix {
		maxPrefix = len(s1)
	}
	if len(s2) < maxPrefix {
		maxPrefix = len(s2)
	}

	for i := 0; i < maxPrefix; i++ {
		if s1[i] == s2[i] {
			prefixLen++
		} else {
			break
		}
	}

	return jaro + float64(prefixLen)*0.1*(1.0-jaro)
}

func jaroSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	len1 := len(s1)
	len2 := len(s2)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	matchDistance := (maxInt(len1, len2) / 2) - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)

	matches := 0
	transpositions := 0

	for i := 0; i < len1; i++ {
		start := maxInt(0, i-matchDistance)
		end := minInt(i+matchDistance+1, len2)

		for j := start; j < end; j++ {
			if s2Matches[j] || s1[i] != s2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := 0; i < len1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if s1[i] != s2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return ((m / float64(len1)) + (m / float64(len2)) + ((m - float64(transpositions)/2.0) / m)) / 3.0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
