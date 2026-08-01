package api

import (
	"strings"

	"github.com/hondyman/uisce/backend/internal/rules"
)

func mapSeverityToContract(s rules.Severity) (SeverityLevel, bool) {
	switch strings.ToUpper(string(s)) {
	case strings.ToUpper(string(rules.SeverityHardBlock)),
		strings.ToUpper(string(rules.SeverityQuarantine)),
		"ERROR":
		return SeverityHardBlock, false
	case strings.ToUpper(string(rules.SeverityWarning)),
		"WARNING":
		return SeveritySoftWarn, true
	default:
		return SeverityInfo, true
	}
}

func rollupHighestSeverity(results []*rules.RuleResult) (highest SeverityLevel, anyHardBlock bool) {
	highest = SeverityInfo
	for _, r := range results {
		level, canOverride := mapSeverityToContract(r.Severity)
		if !canOverride {
			anyHardBlock = true
			highest = level
			return
		}
		if level == SeveritySoftWarn && highest == SeverityInfo {
			highest = level
		}
	}
	return
}
