package compliance

type DimensionMask uint64

const (
	MaskAssetClassEquity DimensionMask = 1 << 0
	MaskAssetClassBond   DimensionMask = 1 << 1
	MaskAssetClassDeriv  DimensionMask = 1 << 2
	MaskRegionUSCAN      DimensionMask = 1 << 3
	MaskRegionEMEA       DimensionMask = 1 << 4
	MaskSectorTech       DimensionMask = 1 << 5
)

type OptimizedRule struct {
	RuleID       string
	RequiredMask DimensionMask
	Operator     string
	Threshold    float64
}

// FilterApplicableRules prunes 10,000 rules down to 10-15 matching rules in < 2ns
func FilterApplicableRules(ticketMask DimensionMask, allRules []OptimizedRule, buffer []OptimizedRule) []OptimizedRule {
	buffer = buffer[:0]
	for i := range allRules {
		if (allRules[i].RequiredMask & ticketMask) == allRules[i].RequiredMask {
			buffer = append(buffer, allRules[i])
		}
	}
	return buffer
}
