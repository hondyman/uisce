package scoring

import (
	"math"
	"sort"
	"strings"
)

// Evaluator executes source scoring, sufficiency calculation, and displacement simulations.
type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateSubstitutionMatrix computes Substitution Rate (SR), Conditional Sufficiency, Solo Rate, and Contribution Share.
func (e *Evaluator) EvaluateSubstitutionMatrix(
	tolerances map[string]AttributeTolerance,
	candidates []VendorCandidate,
	goldenRecords []GoldenRecord,
	overrides []ValueOverrideRecord,
	inScopeUniverseSize int,
) []SubstitutionScore {
	if inScopeUniverseSize <= 0 {
		inScopeUniverseSize = 1
	}

	// Index golden records by (entity_id, attribute_code)
	type entityAttrKey struct {
		EntityID int64
		AttrCode string
	}
	goldenMap := make(map[entityAttrKey]GoldenRecord)
	for _, gr := range goldenRecords {
		goldenMap[entityAttrKey{EntityID: gr.EntityID, AttrCode: gr.AttributeCode}] = gr
	}

	// Index candidate validity to determine Solo records
	// Key: (entity_id, attribute_code) -> slice of valid vendor IDs
	validVendorsPerEntity := make(map[entityAttrKey][]string)
	for _, c := range candidates {
		if c.IsValid() {
			k := entityAttrKey{EntityID: c.EntityID, AttrCode: c.AttributeCode}
			validVendorsPerEntity[k] = append(validVendorsPerEntity[k], c.VendorID)
		}
	}

	// Index overrides to calculate OER per (vendor_id, attribute_code)
	oerMap := e.CalculateOverrideEndorsements(overrides)

	// Group candidates by (vendor_id, attribute_code)
	type vendorAttrKey struct {
		VendorID string
		AttrCode string
	}
	grouped := make(map[vendorAttrKey][]VendorCandidate)
	for _, c := range candidates {
		k := vendorAttrKey{VendorID: c.VendorID, AttrCode: c.AttributeCode}
		grouped[k] = append(grouped[k], c)
	}

	var results []SubstitutionScore

	for k, cList := range grouped {
		tol, hasTol := tolerances[k.AttrCode]
		if !hasTol {
			tol = AttributeTolerance{
				AttributeCode: k.AttrCode,
				Tier:          2,
				MatchType:     MatchExact,
				TierWeight:    0.30,
			}
		}

		availableCount := 0
		validMatchCount := 0
		soloRecordsCount := 0
		goldenWinsCount := 0

		for _, cand := range cList {
			eaKey := entityAttrKey{EntityID: cand.EntityID, AttrCode: cand.AttributeCode}
			gold, hasGold := goldenMap[eaKey]

			if cand.IsValid() {
				availableCount++

				// Check if this vendor is the solo valid provider
				if validList := validVendorsPerEntity[eaKey]; len(validList) == 1 && validList[0] == cand.VendorID {
					soloRecordsCount++
				}

				if hasGold {
					// Check tolerance agreement
					if EvaluateTolerance(tol, cand.NormalizedValue, gold.GoldenValue) {
						validMatchCount++
					}
				}
			}

			if hasGold && strings.EqualFold(gold.WinningVendorID, cand.VendorID) {
				goldenWinsCount++
			}
		}

		inScope := inScopeUniverseSize
		if len(cList) > inScope {
			inScope = len(cList)
		}

		coveragePct := (float64(availableCount) / float64(inScope)) * 100.0
		sufficiencyPct := (float64(validMatchCount) / float64(inScope)) * 100.0
		soloRatePct := (float64(soloRecordsCount) / float64(inScope)) * 100.0
		contributionSharePct := (float64(goldenWinsCount) / float64(inScope)) * 100.0

		var condSufficiency float64
		if availableCount > 0 {
			condSufficiency = (float64(validMatchCount) / float64(availableCount)) * 100.0
		}

		// Retrieve OER if available
		var oerPct float64
		if attrOer, ok := oerMap[k.AttrCode]; ok {
			oerPct = attrOer[k.VendorID]
		}

		results = append(results, SubstitutionScore{
			VendorID:                k.VendorID,
			AttributeCode:           k.AttrCode,
			Tier:                    tol.Tier,
			InScope:                 inScope,
			AvailableCount:          availableCount,
			ValidMatchCount:         validMatchCount,
			CoveragePct:             math.Round(coveragePct*100) / 100,
			SufficiencyRatePct:      math.Round(sufficiencyPct*100) / 100,
			ConditionalSufficiency:  math.Round(condSufficiency*100) / 100,
			SoloRatePct:             math.Round(soloRatePct*100) / 100,
			SoloRecordsCount:        soloRecordsCount,
			ContributionSharePct:    math.Round(contributionSharePct*100) / 100,
			OverrideEndorsementRate: math.Round(oerPct*100) / 100,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Tier != results[j].Tier {
			return results[i].Tier < results[j].Tier
		}
		if results[i].AttributeCode != results[j].AttributeCode {
			return results[i].AttributeCode < results[j].AttributeCode
		}
		return results[i].SufficiencyRatePct > results[j].SufficiencyRatePct
	})

	return results
}

// SimulateVendorDisplacement simulates dropping a vendor from the survivorship hierarchy.
func (e *Evaluator) SimulateVendorDisplacement(
	droppedVendorID string,
	droppedVendorName string,
	annualCost float64,
	tolerances map[string]AttributeTolerance,
	vendorHierarchy []string, // fallback priority order
	candidates []VendorCandidate,
	goldenRecords []GoldenRecord,
) VendorDisplacementResult {
	type entityAttrKey struct {
		EntityID int64
		AttrCode string
	}

	// Index existing golden records
	goldenMap := make(map[entityAttrKey]GoldenRecord)
	for _, gr := range goldenRecords {
		goldenMap[entityAttrKey{EntityID: gr.EntityID, AttrCode: gr.AttributeCode}] = gr
	}

	// Group valid non-dropped candidates by (entity_id, attribute_code)
	candidatePool := make(map[entityAttrKey][]VendorCandidate)
	for _, c := range candidates {
		if strings.EqualFold(c.VendorID, droppedVendorID) {
			continue // exclude dropped vendor
		}
		if c.IsValid() {
			k := entityAttrKey{EntityID: c.EntityID, AttrCode: c.AttributeCode}
			candidatePool[k] = append(candidatePool[k], c)
		}
	}

	// Track outcomes per attribute
	type attrOutcome struct {
		TotalCount int
		Unchanged  int
		Changed    int
		NowNull    int
	}
	outcomes := make(map[string]*attrOutcome)

	for eaKey, oldGold := range goldenMap {
		tol, hasTol := tolerances[eaKey.AttrCode]
		if !hasTol {
			tol = AttributeTolerance{AttributeCode: eaKey.AttrCode, Tier: 2, MatchType: MatchExact}
		}

		out, ok := outcomes[eaKey.AttrCode]
		if !ok {
			out = &attrOutcome{}
			outcomes[eaKey.AttrCode] = out
		}
		out.TotalCount++

		// Find winning candidate among remaining vendors based on priority
		remainingCandidates := candidatePool[eaKey]
		var newWinner *VendorCandidate

		for _, priorityVendor := range vendorHierarchy {
			if strings.EqualFold(priorityVendor, droppedVendorID) {
				continue
			}
			for _, c := range remainingCandidates {
				if strings.EqualFold(c.VendorID, priorityVendor) {
					cCopy := c
					newWinner = &cCopy
					break
				}
			}
			if newWinner != nil {
				break
			}
		}

		if newWinner == nil && len(remainingCandidates) > 0 {
			newWinner = &remainingCandidates[0]
		}

		if newWinner == nil {
			// No other source had this value -> now NULL
			out.NowNull++
		} else {
			// Compare new value with old golden value within tolerance
			if EvaluateTolerance(tol, newWinner.NormalizedValue, oldGold.GoldenValue) {
				out.Unchanged++
			} else {
				out.Changed++
			}
		}
	}

	// Aggregate by tier and residual gaps
	tierAgg := make(map[int]*struct {
		attrCount int
		totalRecs int
		unchanged int
		changed   int
		nowNull   int
	})

	var residualGaps []ResidualGap
	totalNulls := 0

	for attrCode, out := range outcomes {
		tol := tolerances[attrCode]
		tier := tol.Tier
		if tier == 0 {
			tier = 2
		}

		tagg, ok := tierAgg[tier]
		if !ok {
			tagg = &struct {
				attrCount int
				totalRecs int
				unchanged int
				changed   int
				nowNull   int
			}{}
			tierAgg[tier] = tagg
		}
		tagg.attrCount++
		tagg.totalRecs += out.TotalCount
		tagg.unchanged += out.Unchanged
		tagg.changed += out.Changed
		tagg.nowNull += out.NowNull

		if out.NowNull > 0 {
			totalNulls += out.NowNull
			nullPct := (float64(out.NowNull) / float64(out.TotalCount)) * 100.0
			residualGaps = append(residualGaps, ResidualGap{
				AttributeCode: attrCode,
				Tier:          tier,
				SoloRatePct:   math.Round(nullPct*100) / 100,
				RecordsLost:   out.NowNull,
			})
		}
	}

	sort.Slice(residualGaps, func(i, j int) bool {
		return residualGaps[i].RecordsLost > residualGaps[j].RecordsLost
	})

	var tierSummaries []TierDisplacementSummary
	var totalRecordsAllTiers int
	var totalUnchangedAllTiers int

	for tier := 1; tier <= 3; tier++ {
		tagg := tierAgg[tier]
		if tagg == nil || tagg.totalRecs == 0 {
			continue
		}
		totalRecordsAllTiers += tagg.totalRecs
		totalUnchangedAllTiers += tagg.unchanged

		avgInScope := tagg.totalRecs / tagg.attrCount
		tierSummaries = append(tierSummaries, TierDisplacementSummary{
			Tier:            tier,
			AttributesCount: tagg.attrCount,
			AverageInScope:  avgInScope,
			UnchangedPct:    math.Round((float64(tagg.unchanged)/float64(tagg.totalRecs))*1000) / 10,
			ChangedPct:      math.Round((float64(tagg.changed)/float64(tagg.totalRecs))*1000) / 10,
			NowNullPct:      math.Round((float64(tagg.nowNull)/float64(tagg.totalRecs))*1000) / 10,
		})
	}

	var displacementReadiness float64
	if totalRecordsAllTiers > 0 {
		displacementReadiness = (float64(totalUnchangedAllTiers) / float64(totalRecordsAllTiers)) * 100.0
	}

	// Cost estimates: $18 standard ops remediation per unreplaced critical field
	remediationCostEst := float64(totalNulls) * 18.0
	netFirstYearBenefit := annualCost - remediationCostEst

	var costPerUniqueVal float64
	if totalNulls > 0 {
		costPerUniqueVal = annualCost / float64(totalNulls)
	}

	return VendorDisplacementResult{
		DroppedVendorID:       droppedVendorID,
		DroppedVendorName:     droppedVendorName,
		AnnualCost:            annualCost,
		DisplacementReadiness: math.Round(displacementReadiness*10) / 10,
		TierSummaries:         tierSummaries,
		ResidualGaps:          residualGaps,
		TotalNullValues:       totalNulls,
		RemediationCostEst:    remediationCostEst,
		NetFirstYearBenefit:   netFirstYearBenefit,
		CostPerUniqueValue:    math.Round(costPerUniqueVal*100) / 100,
	}
}

// CalculateOverrideEndorsements computes OER: attribute -> vendor -> pct
func (e *Evaluator) CalculateOverrideEndorsements(overrides []ValueOverrideRecord) map[string]map[string]float64 {
	totalOverridesPerAttr := make(map[string]int)
	vendorOverridesPerAttr := make(map[string]map[string]int)

	for _, o := range overrides {
		if o.EndorsementVendorID == "" {
			continue
		}
		totalOverridesPerAttr[o.AttributeCode]++
		if _, ok := vendorOverridesPerAttr[o.AttributeCode]; !ok {
			vendorOverridesPerAttr[o.AttributeCode] = make(map[string]int)
		}
		vendorOverridesPerAttr[o.AttributeCode][o.EndorsementVendorID]++
	}

	result := make(map[string]map[string]float64)
	for attr, vCounts := range vendorOverridesPerAttr {
		total := totalOverridesPerAttr[attr]
		if total == 0 {
			continue
		}
		result[attr] = make(map[string]float64)
		for vID, count := range vCounts {
			result[attr][vID] = (float64(count) / float64(total)) * 100.0
		}
	}
	return result
}

// ComputeValueForMoneyFrontier identifies Pareto-optimal vendors on the cost/quality plane.
func (e *Evaluator) ComputeValueForMoneyFrontier(
	vendors map[string]string, // id -> name
	costs map[string]float64, // id -> annual fee
	scores map[string]float64, // id -> quality index 0-100
) []ValueForMoneyPoint {
	var points []ValueForMoneyPoint

	for id, name := range vendors {
		cost := costs[id]
		score := scores[id]

		points = append(points, ValueForMoneyPoint{
			VendorID:     id,
			VendorName:   name,
			AnnualCost:   cost,
			QualityIndex: math.Round(score*100) / 100,
		})
	}

	// Sort by ascending cost to compute non-dominated convex hull
	sort.Slice(points, func(i, j int) bool {
		return points[i].AnnualCost < points[j].AnnualCost
	})

	var maxScoreSoFar float64 = -1.0
	for i := range points {
		if points[i].QualityIndex > maxScoreSoFar {
			points[i].IsOnFrontier = true
			maxScoreSoFar = points[i].QualityIndex
		} else {
			points[i].IsOnFrontier = false
		}
	}

	// Baseline quality without this vendor
	for i := range points {
		if points[i].QualityIndex > 0 {
			points[i].CostPerQualityPoint = math.Round(points[i].AnnualCost / points[i].QualityIndex)
		}
	}

	return points
}
