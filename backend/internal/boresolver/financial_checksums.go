package boresolver

import (
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// ValidateISIN verifies the ISO 6166 International Securities Identification Number using the Luhn mod-10 algorithm.
func ValidateISIN(isin string) bool {
	isin = strings.TrimSpace(strings.ToUpper(isin))
	if len(isin) != 12 {
		return false
	}

	// Country code: first 2 characters must be letters
	if !unicode.IsLetter(rune(isin[0])) || !unicode.IsLetter(rune(isin[1])) {
		return false
	}

	// Last char must be a digit
	if !unicode.IsDigit(rune(isin[11])) {
		return false
	}

	// Convert letters to numbers (A=10, B=11, ..., Z=35)
	var digitStr strings.Builder
	for i := 0; i < 11; i++ {
		ch := rune(isin[i])
		if unicode.IsLetter(ch) {
			val := int(ch - 'A' + 10)
			digitStr.WriteString(strconv.Itoa(val))
		} else if unicode.IsDigit(ch) {
			digitStr.WriteRune(ch)
		} else {
			return false
		}
	}

	s := digitStr.String()
	sum := 0
	double := true

	// Luhn check from right to left
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d = (d / 10) + (d % 10)
			}
		}
		sum += d
		double = !double
	}

	expectedCheckDigit := (10 - (sum % 10)) % 10
	actualCheckDigit := int(isin[11] - '0')

	return expectedCheckDigit == actualCheckDigit
}

// ValidateCUSIP verifies the 9-character North American security identifier using Mod-10.
func ValidateCUSIP(cusip string) bool {
	cusip = strings.TrimSpace(strings.ToUpper(cusip))
	if len(cusip) != 9 {
		return false
	}

	if !unicode.IsDigit(rune(cusip[8])) {
		return false
	}

	sum := 0
	for i := 0; i < 8; i++ {
		ch := rune(cusip[i])
		var val int
		if unicode.IsDigit(ch) {
			val = int(ch - '0')
		} else if unicode.IsLetter(ch) {
			val = int(ch - 'A' + 10)
		} else if ch == '*' {
			val = 36
		} else if ch == '@' {
			val = 37
		} else if ch == '#' {
			val = 38
		} else {
			return false
		}

		if i%2 == 1 { // Even position (2nd, 4th, 6th, 8th) -> 0-indexed odd
			val *= 2
		}

		sum += (val / 10) + (val % 10)
	}

	expectedCheckDigit := (10 - (sum % 10)) % 10
	actualCheckDigit := int(cusip[8] - '0')

	return expectedCheckDigit == actualCheckDigit
}

// ValidateSEDOL verifies the 7-character UK/Ireland security identifier using weighted Mod-10.
func ValidateSEDOL(sedol string) bool {
	sedol = strings.TrimSpace(strings.ToUpper(sedol))
	if len(sedol) != 7 {
		return false
	}

	if !unicode.IsDigit(rune(sedol[6])) {
		return false
	}

	weights := []int{1, 3, 1, 7, 3, 9}
	sum := 0

	for i := 0; i < 6; i++ {
		ch := rune(sedol[i])
		var val int
		if unicode.IsDigit(ch) {
			val = int(ch - '0')
		} else if unicode.IsLetter(ch) {
			// SEDOL excludes vowels in practice, but letter value is A=10..Z=35
			val = int(ch - 'A' + 10)
		} else {
			return false
		}

		sum += val * weights[i]
	}

	expectedCheckDigit := (10 - (sum % 10)) % 10
	actualCheckDigit := int(sedol[6] - '0')

	return expectedCheckDigit == actualCheckDigit
}

// ValidateLEI verifies the 20-character ISO 17442 Legal Entity Identifier using ISO 7064 Mod 97-10.
func ValidateLEI(lei string) bool {
	lei = strings.TrimSpace(strings.ToUpper(lei))
	if len(lei) != 20 {
		return false
	}

	var digitStr strings.Builder
	for _, ch := range lei {
		if unicode.IsLetter(ch) {
			val := int(ch - 'A' + 10)
			digitStr.WriteString(strconv.Itoa(val))
		} else if unicode.IsDigit(ch) {
			digitStr.WriteRune(ch)
		} else {
			return false
		}
	}

	// Compute ISO 7064 Mod 97-10 using big.Int
	n := new(big.Int)
	n, ok := n.SetString(digitStr.String(), 10)
	if !ok {
		return false
	}

	mod97 := new(big.Int).Mod(n, big.NewInt(97))
	return mod97.Int64() == 1
}
