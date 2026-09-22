package dynamic

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
)

var identifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*(\.[a-z_][a-z0-9_]*)?$`)
var aliasPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
var literalPattern = regexp.MustCompile(`^[a-zA-Z0-9 _:.\-/,]*$`)

func SafeIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("empty identifier")
	}
	if !identifierPattern.MatchString(s) {
		return fmt.Errorf("identifier fails safe pattern")
	}
	return nil
}

func SafeAlias(s string) error {
	if s == "" {
		return fmt.Errorf("empty alias")
	}
	if !aliasPattern.MatchString(s) {
		return fmt.Errorf("alias fails safe pattern")
	}
	return nil
}

func NormalizeAndValidateSource(table, column string) (string, string, error) {
	rawTable, rawColumn := table, column
	table = strings.ToLower(strings.TrimSpace(table))
	column = strings.ToLower(strings.TrimSpace(column))

	if table == "" || column == "" {
		log.Printf("dynamic: rejected empty source (rawTable=%q rawColumn=%q)", rawTable, rawColumn)
		return "", "", fmt.Errorf("table and column are required")
	}
	if err := SafeIdentifier(table); err != nil {
		log.Printf("dynamic: rejected unsafe table identifier %q", rawTable)
		return "", "", fmt.Errorf("table identifier is invalid")
	}
	if err := SafeIdentifier(column); err != nil {
		log.Printf("dynamic: rejected unsafe column identifier %q", rawColumn)
		return "", "", fmt.Errorf("column identifier is invalid")
	}

	cols, ok := AllowedEnumSources[table]
	if !ok {
		log.Printf("dynamic: rejected unknown source table %q", rawTable)
		return "", "", fmt.Errorf("table is not in the dynamic-introspection allow-list")
	}
	for _, c := range cols {
		if c == column {
			return table, column, nil
		}
	}
	log.Printf("dynamic: rejected column %q on table %q", rawColumn, rawTable)
	return "", "", fmt.Errorf("column is not allow-listed for the requested table")
}

func ValidateTable(table string) (string, error) {
	raw := table
	table = strings.ToLower(strings.TrimSpace(table))
	if table == "" {
		log.Printf("dynamic: rejected empty table")
		return "", fmt.Errorf("table is required")
	}
	if err := SafeIdentifier(table); err != nil {
		log.Printf("dynamic: rejected unsafe table %q", raw)
		return "", fmt.Errorf("table identifier is invalid")
	}
	if _, ok := AllowedEnumSources[table]; !ok {
		log.Printf("dynamic: rejected unknown table %q", raw)
		return "", fmt.Errorf("table is not in the dynamic-introspection allow-list")
	}
	return table, nil
}

func NormalizeAndValidateAlias(alias string) (string, error) {
	raw := alias
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" {
		return "", nil
	}
	if err := SafeAlias(alias); err != nil {
		log.Printf("dynamic: rejected unsafe alias %q", raw)
		return "", fmt.Errorf("alias identifier is invalid")
	}
	return alias, nil
}

func InSet(value, label string, allowed map[string]bool) error {
	raw := value
	value = strings.TrimSpace(value)
	if !allowed[value] {
		log.Printf("dynamic: rejected %s %q", label, raw)
		return fmt.Errorf("%s is not in the allow-list", label)
	}
	return nil
}

var allowedSeparatorRunes = map[rune]bool{
	'-': true, '/': true, ':': true, '.': true, ',': true, ' ': true, 'T': true,
}

func SafeFormatString(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var token, sep strings.Builder

	checkToken := func() error {
		if token.Len() == 0 {
			return nil
		}
		up := strings.ToUpper(token.String())
		token.Reset()
		if !AllowedDateFormatTokens[up] {
			log.Printf("dynamic: rejected unsupported format element %q (raw=%q)", up, s)
			return fmt.Errorf("format contains an unsupported element")
		}
		return nil
	}
	checkSep := func() error {
		if sep.Len() == 0 {
			return nil
		}
		run := sep.String()
		sep.Reset()
		for _, r := range run {
			if !allowedSeparatorRunes[r] {
				log.Printf("dynamic: rejected unsafe format separator %q (raw=%q)", run, s)
				return fmt.Errorf("format contains an unsupported separator")
			}
		}
		return nil
	}

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if err := checkSep(); err != nil {
				return err
			}
			token.WriteRune(r)
		} else {
			if err := checkToken(); err != nil {
				return err
			}
			sep.WriteRune(r)
		}
	}
	if err := checkToken(); err != nil {
		return err
	}
	return checkSep()
}

func SafeLiteral(s string) error {
	raw := s
	if !literalPattern.MatchString(s) {
		log.Printf("dynamic: rejected unsafe literal %q", raw)
		return fmt.Errorf("literal contains an unsupported character")
	}
	return nil
}
