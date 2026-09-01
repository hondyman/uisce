package mdm

import (
	"strings"
	"unicode"
)

type SymbologyValidator struct{}

func NewSymbologyValidator() *SymbologyValidator {
	return &SymbologyValidator{}
}

// ValidateISIN validates ISO 6166 12-character ISIN strings via modified Luhn algorithm
func (v *SymbologyValidator) ValidateISIN(isin string) bool {
	isin = strings.TrimSpace(strings.ToUpper(isin))
	if len(isin) != 12 {
		return false
	}

	if !unicode.IsLetter(rune(isin[0])) || !unicode.IsLetter(rune(isin[1])) {
		return false
	}

	var digits [30]int
	pos := 0
	for i := 0; i < 12; i++ {
		c := isin[i]
		if unicode.IsDigit(rune(c)) {
			digits[pos] = int(c - '0')
			pos++
		} else if unicode.IsLetter(rune(c)) {
			val := int(c - 'A' + 10)
			digits[pos] = val / 10
			digits[pos+1] = val % 10
			pos += 2
		} else {
			return false
		}
	}

	sum := 0
	double := true
	for i := pos - 2; i >= 0; i-- {
		d := digits[i]
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}

	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit == digits[pos-1]
}

// ValidateLEI validates ISO 7064 Mod 97-10 20-character Legal Entity Identifiers
func (v *SymbologyValidator) ValidateLEI(lei string) bool {
	lei = strings.TrimSpace(strings.ToUpper(lei))
	if len(lei) != 20 {
		return false
	}

	remainder := 0
	for i := 0; i < 20; i++ {
		c := lei[i]
		var val int
		if unicode.IsDigit(rune(c)) {
			val = int(c - '0')
			remainder = (remainder*10 + val) % 97
		} else if unicode.IsLetter(rune(c)) {
			val = int(c - 'A' + 10)
			remainder = (remainder*100 + val) % 97
		} else {
			return false
		}
	}

	return remainder == 1
}
