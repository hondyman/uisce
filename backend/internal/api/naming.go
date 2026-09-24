package api

import (
	"regexp"
	"strings"
)

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

var addrLineRe = regexp.MustCompile(`^(?:address_)?line_?([0-9]+)$`)

func tokenizeColumnName(name string) []string {
	spaced := camelBoundary.ReplaceAllString(name, "$1 $2")
	spaced = strings.NewReplacer("_", " ", ".", " ", "-", " ", "/", " ").Replace(spaced)
	var tokens []string
	for _, t := range strings.Fields(spaced) {
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

var commonShortWords = map[string]bool{
	"NO": true, "OF": true, "IN": true, "ON": true, "AT": true,
	"IS": true, "OR": true, "TO": true, "BY": true, "AN": true, "UP": true,
	"DUE": true, "NEW": true, "OLD": true, "KEY": true, "PIN": true,
}

func looksLikeAbbreviation(token string) bool {
	upper := strings.ToUpper(token)
	if commonShortWords[upper] {
		return false
	}
	if token == strings.ToUpper(token) && len(token) <= 6 {
		return true
	}
	return false
}

func titleCase(tokens []string) string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if len(t) == 1 {
			out = append(out, strings.ToUpper(t))
			continue
		}
		out = append(out, strings.ToUpper(t[:1])+strings.ToLower(t[1:]))
	}
	return strings.Join(out, " ")
}

func pascalCase(tokens []string) string {
	var b strings.Builder
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if t == strings.ToUpper(t) && len(t) <= 3 {
			b.WriteString(t)
			continue
		}
		b.WriteString(strings.ToUpper(t[:1]) + strings.ToLower(t[1:]))
	}
	return b.String()
}

func pascalCaseToWords(s string) []string {
	var words []string
	var current, acronym strings.Builder
	flushCurrent := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	flushAcronym := func() {
		if acronym.Len() > 0 {
			words = append(words, acronym.String())
			acronym.Reset()
		}
	}

	prevUpperFollowedByLower := false

	for i := 0; i < len(s); i++ {
		r := rune(s[i])
		if r >= 'A' && r <= 'Z' {
			nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
			nextUpper := i+1 < len(s) && s[i+1] >= 'A' && s[i+1] <= 'Z'
			prevUpper := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'

			if nextLower {
				flushAcronym()
				flushCurrent()
				current.WriteRune(r)
			} else if prevUpper && !prevUpperFollowedByLower && nextUpper {
				current.WriteRune(r)
			} else {
				flushAcronym()
				current.WriteRune(r)
			}

			prevUpperFollowedByLower = nextLower
		} else {
			flushAcronym()
			current.WriteRune(r)
			prevUpperFollowedByLower = false
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	if acronym.Len() > 0 {
		words = append(words, acronym.String())
	}
	if len(words) == 0 {
		words = []string{s}
	}
	return words
}

var genericWords = map[string]bool{
	"city": true, "status": true, "name": true, "type": true, "code": true,
	"number": true, "amount": true, "balance": true, "rate": true, "date": true,
	"id": true, "value": true, "desc": true, "level": true, "group": true,
	"class": true, "category": true, "source": true, "target": true,
}

func isGenericWord(word string) bool {
	return genericWords[strings.ToLower(word)]
}

func extractTableNameFromContext(tableSchemaContext string) string {
	parts := strings.Split(tableSchemaContext, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		part = strings.Trim(part, "\"")
		if part != "" && !strings.HasPrefix(part, "table ") && !strings.HasPrefix(part, "schema ") {
			return part
		}
	}
	re := regexp.MustCompile(`table\s+"([^"]+)"`)
	m := re.FindStringSubmatch(tableSchemaContext)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

type derivedTermNames struct {
	SemanticName    string
	BusinessName    string
	BaseGenericTerm string
	ContextSensitive bool   // true iff any derivation rule consulted table/schema context
	source          string // one of: "pascal", "abbrev_map", "addr_line_context", "bare_generic"
}

func (d derivedTermNames) GetSource() string {
	return d.source
}

// CandidateTerm is a single candidate in the ranked candidate list returned by
// deriveTermNamesCandidates. Rank 0 = primary suggestion; rank N = Nth fallback.
type CandidateTerm struct {
	Name   string
	Source string
}

// deriveTermNamesCandidates returns a ranked list of candidate semantic term names,
// ordered from most preferred to least. It is deterministic-only (no LLM) and
// is used by both the preview endpoint and the generate path to implement
// "skip rejected candidates" logic.
//
// The ranked order is:
//   1. SemanticName (contextual PascalCase, e.g. EmployeeAddress1)
//   2. BusinessName (title-case, e.g. Employee Address Line 1) — only included
//      if different from SemanticName
//   3. BaseGenericTerm qualified with table context — only present when the
//      address-line rule applied (e.g. Address for address_line_N without a table)
//   4. Naive PascalCase on the original tokens — always present as final fallback
//
// Rejection filtering (caller's responsibility): iterate candidates in order,
// pick the first one not in the rejection set; if all are rejected, use the
// naive PascalCase fallback with source "pascal".
func deriveTermNamesCandidates(resolvedTokens []string, rawName string, tableSchemaContext string) []CandidateTerm {
	if len(resolvedTokens) == 0 {
		return nil
	}

	derived := deriveTermNamesDeterministic(resolvedTokens, rawName, tableSchemaContext)

	var candidates []CandidateTerm

	// 1. Primary: semantic name (always present)
	candidates = append(candidates, CandidateTerm{Name: derived.SemanticName, Source: derived.source})

	// 2. Business name (only if different from semantic — prevents duplicate candidates)
	if derived.BusinessName != "" && derived.BusinessName != derived.SemanticName {
		candidates = append(candidates, CandidateTerm{Name: derived.BusinessName, Source: derived.source})
	}

	// 3. Base generic term: for address_line_N without a table qualifier, the bare
	//    "Address" is a legitimate fallback the user might prefer over the contextual
	//    rule's output. Only included when the address-line rule applied AND the
	//    base term is non-empty AND it's different from both semantic and business names.
	if derived.BaseGenericTerm != "" {
		baseTerm := derived.BaseGenericTerm
		if baseTerm != derived.SemanticName && baseTerm != derived.BusinessName {
			candidates = append(candidates, CandidateTerm{Name: baseTerm, Source: "bare_generic"})
		}
	}

	// 4. Naive PascalCase fallback: always last resort. It is built from the ORIGINAL
	// column tokens, not the abbreviation-expanded ones: from the resolved tokens it
	// duplicates the primary (auditor_id -> AuditorIdentifier), is de-duplicated away,
	// and leaves the candidate list exhausted after one rejection.
	naiveTokens := tokenizeColumnName(rawName)
	if len(naiveTokens) == 0 {
		naiveTokens = resolvedTokens
	}
	naive := pascalCase(naiveTokens)
	if naive != derived.SemanticName && naive != derived.BusinessName && naive != derived.BaseGenericTerm {
		candidates = append(candidates, CandidateTerm{Name: naive, Source: "pascal"})
	}

	return candidates
}

// deriveTermNamesDeterministic applies the contextual naming rules to already-resolved
// tokens. It does NOT resolve abbreviations or call the LLM — those steps must
// happen before this is called. Contract:
//   - resolvedTokens: output of abbreviation-map resolution; each token is already
//     expanded to its full form (e.g. ["ACKNOWLEDGED", "line", "1"])
//   - rawName: original column name used for pattern matching (address_line_1, ack_at)
//   - tableSchemaContext: used only for the address-line-N contextual qualifier
//
// The function applies: address-line contextualization, bare generic-word base-term
// detection, and PascalCase/titleCase assembly. All three are pure and deterministic.
func deriveTermNamesDeterministic(resolvedTokens []string, rawName string, tableSchemaContext string) derivedTermNames {
	if len(resolvedTokens) == 0 {
		return derivedTermNames{}
	}

	semanticName := pascalCase(resolvedTokens)
	businessName := titleCase(resolvedTokens)
	var baseGenericTerm string
	contextSensitive := false

	tableName := extractTableNameFromContext(tableSchemaContext)

	if addrMatch := addrLineRe.FindStringSubmatch(strings.ToLower(rawName)); addrMatch != nil && tableName != "" {
		contextSensitive = true // address-line rule consults table name
		lineNum := addrMatch[1]
		tableTokens := tokenizeColumnName(tableName)
		var entityParts []string
		for _, tt := range tableTokens {
			if !strings.EqualFold(tt, "address") {
				entityParts = append(entityParts, strings.Title(strings.ToLower(tt)))
			}
		}
		if len(entityParts) > 0 {
			semanticName = strings.Join(entityParts, "") + "Address" + lineNum
			businessName = strings.Join(entityParts, " ") + " Address Line " + lineNum
		} else {
			semanticName = "Address" + lineNum
			businessName = "Address Line " + lineNum
		}
		baseGenericTerm = "Address"
	} else if len(resolvedTokens) == 1 && strings.EqualFold(resolvedTokens[0], rawName) && isGenericWord(resolvedTokens[0]) && tableSchemaContext != "" {
		if tableName != "" && !strings.EqualFold(tableName, rawName) {
			contextSensitive = true // bare-generic rule consults table name
			baseGenericTerm = strings.Title(strings.ToLower(resolvedTokens[0]))
		}
	}

	source := "pascal"
	if addrMatch := addrLineRe.FindStringSubmatch(strings.ToLower(rawName)); addrMatch != nil && tableName != "" {
		source = "addr_line_context"
	} else if len(resolvedTokens) == 1 && strings.EqualFold(resolvedTokens[0], rawName) && isGenericWord(resolvedTokens[0]) && tableSchemaContext != "" && tableName != "" && !strings.EqualFold(tableName, rawName) {
		source = "bare_generic"
	}

	return derivedTermNames{
		SemanticName:     semanticName,
		BusinessName:     businessName,
		BaseGenericTerm:  baseGenericTerm,
		ContextSensitive: contextSensitive,
		source:           source,
	}
}

// sanitizeExpansion makes an abbreviation expansion safe to build a term name from. An LLM may answer with
// several words or with separators ("Service_level_agreement"), which ended up verbatim inside term names
// ("Service_level_agreementMet"). Split on anything that is not a letter or digit and title-case each word, so
// the name is plain PascalCase. A single clean word is returned as it was given.
func sanitizeExpansion(full string) string {
	words := strings.FieldsFunc(full, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	if len(words) == 0 {
		return ""
	}
	if len(words) == 1 {
		return words[0]
	}
	return pascalCase(words)
}
