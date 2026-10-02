package attribute

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	safeIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	safeFieldCd = regexp.MustCompile(`^[a-z][a-z0-9_]{0,99}$`)
)

func quoteIdent(name string) (string, error) {
	if !safeIdentRe.MatchString(name) {
		return "", fmt.Errorf("unsafe identifier: %q", name)
	}
	return `"` + name + `"`, nil
}

func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func sanitizeTableRef(ref string) (schema, table string, err error) {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "crims.")
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("table_ref must be schema.table, got %q", ref)
	}
	if !safeIdentRe.MatchString(parts[0]) || !safeIdentRe.MatchString(parts[1]) {
		return "", "", fmt.Errorf("unsafe table_ref: %q", ref)
	}
	return parts[0], parts[1], nil
}

func safeTableSQL(ref string) (string, error) {
	schema, table, err := sanitizeTableRef(ref)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s"."%s"`, schema, table), nil
}

func normalizeFieldCd(name, fieldCd string) (string, error) {
	if fieldCd != "" {
		fieldCd = strings.ToLower(strings.TrimSpace(fieldCd))
		if !safeFieldCd.MatchString(fieldCd) {
			return "", fmt.Errorf("field_cd must be snake_case starting with a letter")
		}
		return fieldCd, nil
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if !safeFieldCd.MatchString(out) {
		return "", fmt.Errorf("could not derive field_cd from name %q", name)
	}
	return out, nil
}

func entityTypeFromTable(table string) string {
	return strings.ToUpper(table)
}

func buildCustomColumnExpr(tblAlias string, d AttributeDef) string {
	key := fmt.Sprintf("%s.custom_attributes->>%s", tblAlias, quoteLiteral(d.FieldCd))
	switch strings.ToLower(d.DataType) {
	case "integer":
		return fmt.Sprintf("mdm.safe_int(%s)", key)
	case "decimal", "numeric", "number":
		return fmt.Sprintf("mdm.safe_numeric(%s)", key)
	case "boolean", "bool":
		return fmt.Sprintf("mdm.safe_bool(%s)", key)
	case "date":
		return fmt.Sprintf("mdm.safe_date(%s)", key)
	case "json", "object", "array":
		return fmt.Sprintf(
			"CASE WHEN %s.custom_attributes ? %s THEN %s.custom_attributes->%s ELSE NULL END",
			tblAlias, quoteLiteral(d.FieldCd),
			tblAlias, quoteLiteral(d.FieldCd),
		)
	default:
		return key
	}
}

func humanLabel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
