package engine

import (
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// computeOwnershipAndDisclosureMetrics calculates firm-wide voting equity, takeover control, and net short position metrics.
func computeOwnershipAndDisclosureMetrics(state *PortfolioState, metrics map[string]decimal.Decimal) {
	// Build adjacency list for hierarchy edges, deterministically sorted by ChildAccountID
	adjMap := make(map[uuid.UUID][]FundHierarchyEdge)
	for _, edge := range state.HierarchyEdges {
		adjMap[edge.ParentAccountID] = append(adjMap[edge.ParentAccountID], edge)
	}
	for parentID := range adjMap {
		edges := adjMap[parentID]
		sort.Slice(edges, func(i, j int) bool {
			return edges[i].ChildAccountID.String() < edges[j].ChildAccountID.String()
		})
		adjMap[parentID] = edges
	}

	// 1. Voting Equity & Control Aggregation (Rule 37 SEC 13D, Rule 38 UK DTR5, Rule 39 EU Transparency, Rule 40 Takeover)
	var maxFirmwideVotingEquityPct decimal.Decimal
	var maxTakeoverVotingControlPct decimal.Decimal

	// Group holdings by IssuerID
	issuerVotingMap := make(map[string]decimal.Decimal)
	issuerTotalSharesMap := make(map[string]decimal.Decimal)
	issuerTakeoverMap := make(map[string]decimal.Decimal)

	for _, pos := range state.Positions {
		if pos.IssuerID == "" {
			continue
		}

		// Direct position voting power calculation
		if pos.TotalVotingPowerPct.GreaterThan(decimal.Zero) {
			issuerVotingMap[pos.IssuerID] = issuerVotingMap[pos.IssuerID].Add(pos.TotalVotingPowerPct)
		} else if pos.VotingSharesOutstanding.GreaterThan(decimal.Zero) && pos.VotingSharesHeld.GreaterThan(decimal.Zero) {
			votingPct := pos.VotingSharesHeld.Div(pos.VotingSharesOutstanding)
			issuerVotingMap[pos.IssuerID] = issuerVotingMap[pos.IssuerID].Add(votingPct)
		} else if pos.SharesOutstanding.GreaterThan(decimal.Zero) && pos.SharesHeld.GreaterThan(decimal.Zero) {
			votingPct := pos.SharesHeld.Div(pos.SharesOutstanding)
			issuerVotingMap[pos.IssuerID] = issuerVotingMap[pos.IssuerID].Add(votingPct)
		}

		// Takeover Target Voting Control
		if pos.IsTakeoverTarget {
			if pos.TotalVotingPowerPct.GreaterThan(decimal.Zero) {
				issuerTakeoverMap[pos.IssuerID] = issuerTakeoverMap[pos.IssuerID].Add(pos.TotalVotingPowerPct)
			} else if pos.VotingSharesOutstanding.GreaterThan(decimal.Zero) && pos.VotingSharesHeld.GreaterThan(decimal.Zero) {
				takeoverPct := pos.VotingSharesHeld.Div(pos.VotingSharesOutstanding)
				issuerTakeoverMap[pos.IssuerID] = issuerTakeoverMap[pos.IssuerID].Add(takeoverPct)
			} else if pos.SharesOutstanding.GreaterThan(decimal.Zero) && pos.SharesHeld.GreaterThan(decimal.Zero) {
				takeoverPct := pos.SharesHeld.Div(pos.SharesOutstanding)
				issuerTakeoverMap[pos.IssuerID] = issuerTakeoverMap[pos.IssuerID].Add(takeoverPct)
			}
		}

		if pos.SharesOutstanding.GreaterThan(decimal.Zero) {
			issuerTotalSharesMap[pos.IssuerID] = pos.SharesOutstanding
		}
	}

	// Traverse hierarchy graph for child account holdings with cycle-pruning
	if len(adjMap) > 0 {
		activePath := make(map[uuid.UUID]bool)
		traverseHierarchy(state.AccountID, adjMap, activePath, decimal.RequireFromString("1.000000"), decimal.RequireFromString("1.000000"), issuerVotingMap, issuerTakeoverMap)
	}

	for _, v := range issuerVotingMap {
		if v.GreaterThan(maxFirmwideVotingEquityPct) {
			maxFirmwideVotingEquityPct = v
		}
	}
	for _, v := range issuerTakeoverMap {
		if v.GreaterThan(maxTakeoverVotingControlPct) {
			maxTakeoverVotingControlPct = v
		}
	}

	metrics["portfolio.firmwide_equity_voting_pct"] = maxFirmwideVotingEquityPct
	metrics["portfolio.firmwide_voting_control_pct"] = maxTakeoverVotingControlPct

	// 2. Net Short Position Aggregation (Rule 41 EU SSR, Rule 42 UK SSR)
	// Net Short = (Gross Short Shares + Delta Equiv Short) - (Gross Long Shares + Delta Equiv Long) / Shares Outstanding
	var maxFirmwideNetShortPct decimal.Decimal

	issuerNetShortMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.IssuerID == "" {
			continue
		}
		sharesOut := pos.SharesOutstanding
		if sharesOut.IsZero() {
			sharesOut = pos.VotingSharesOutstanding
		}

		if sharesOut.GreaterThan(decimal.Zero) {
			totalShortShares := pos.GrossShortShares.Add(pos.DeltaEquivShortShares)
			totalLongShares := pos.GrossLongShares.Add(pos.DeltaEquivLongShares)
			netShortShares := totalShortShares.Sub(totalLongShares)

			if netShortShares.GreaterThan(decimal.Zero) {
				netShortPct := netShortShares.Div(sharesOut)
				issuerNetShortMap[pos.IssuerID] = issuerNetShortMap[pos.IssuerID].Add(netShortPct)
			} else {
				// Net long or flat position -> 0 net short exposure
				if _, exists := issuerNetShortMap[pos.IssuerID]; !exists {
					issuerNetShortMap[pos.IssuerID] = decimal.Zero
				}
			}
		}
	}

	for _, netShort := range issuerNetShortMap {
		if netShort.GreaterThan(maxFirmwideNetShortPct) {
			maxFirmwideNetShortPct = netShort
		}
	}
	metrics["portfolio.firmwide_net_short_pct"] = maxFirmwideNetShortPct

	// 3. Prior Snapshot Metric Provenance & Delta Crossing Detection
	if state.PriorSnapshotMetrics != nil {
		if priorVoting, ok := state.PriorSnapshotMetrics["portfolio.firmwide_equity_voting_pct"]; ok {
			metrics["portfolio.prior_firmwide_equity_voting_pct"] = priorVoting
		} else {
			metrics["portfolio.prior_firmwide_equity_voting_pct"] = decimal.Zero
		}
		if priorTakeover, ok := state.PriorSnapshotMetrics["portfolio.firmwide_voting_control_pct"]; ok {
			metrics["portfolio.prior_firmwide_voting_control_pct"] = priorTakeover
		} else {
			metrics["portfolio.prior_firmwide_voting_control_pct"] = decimal.Zero
		}
		if priorNetShort, ok := state.PriorSnapshotMetrics["portfolio.firmwide_net_short_pct"]; ok {
			metrics["portfolio.prior_firmwide_net_short_pct"] = priorNetShort
		} else {
			metrics["portfolio.prior_firmwide_net_short_pct"] = decimal.Zero
		}
	} else {
		// First-ever evaluation from zero
		metrics["portfolio.prior_firmwide_equity_voting_pct"] = decimal.Zero
		metrics["portfolio.prior_firmwide_voting_control_pct"] = decimal.Zero
		metrics["portfolio.prior_firmwide_net_short_pct"] = decimal.Zero
	}

	// 4. ERISA Benefit Plan Investor (BPI) 25% Significant Participation Calculation
	var erisaBpiEquityPct decimal.Decimal
	if state.ErisaBpiEquityPct.GreaterThan(decimal.Zero) {
		erisaBpiEquityPct = state.ErisaBpiEquityPct
	} else if len(state.Investors) > 0 {
		var totalBPIEquity, totalFundEquity, disregardedGPEquity decimal.Decimal
		for _, inv := range state.Investors {
			totalFundEquity = totalFundEquity.Add(inv.EquityValue)
			if inv.IsGPDisregarded || inv.InvestorType == "GP_MANAGEMENT" {
				disregardedGPEquity = disregardedGPEquity.Add(inv.EquityValue)
			} else if inv.IsBenefitPlan || inv.InvestorType == "ERISA_BENEFIT_PLAN" || inv.InvestorType == "IRA_INDIVIDUAL" || inv.InvestorType == "PLAN_ASSET_ENTITY" {
				totalBPIEquity = totalBPIEquity.Add(inv.EquityValue)
			}
		}
		eligibleEquity := totalFundEquity.Sub(disregardedGPEquity)
		if eligibleEquity.GreaterThan(decimal.Zero) {
			erisaBpiEquityPct = totalBPIEquity.Div(eligibleEquity)
		}
	}
	metrics["portfolio.erisa_bpi_equity_pct"] = erisaBpiEquityPct

	// 5. Passive Investor Intent for SEC Schedule 13G
	isPassive := decimal.RequireFromString("1.000000") // default true (1.0)
	if state.IsPassiveIntent != nil && !*state.IsPassiveIntent {
		isPassive = decimal.Zero // false (0.0)
	}
	metrics["portfolio.is_passive_intent"] = isPassive
}

// traverseHierarchy performs deterministic DFS traversal with active-branch cycle pruning.
func traverseHierarchy(
	currentID uuid.UUID,
	adjMap map[uuid.UUID][]FundHierarchyEdge,
	activePath map[uuid.UUID]bool,
	currentVotingWeight decimal.Decimal,
	currentEconomicWeight decimal.Decimal,
	issuerVotingMap map[string]decimal.Decimal,
	issuerTakeoverMap map[string]decimal.Decimal,
) {
	if activePath[currentID] {
		// Cycle detected on active branch path: prune and halt descent
		return
	}

	activePath[currentID] = true
	defer func() {
		delete(activePath, currentID)
	}()

	edges, ok := adjMap[currentID]
	if !ok {
		return
	}

	for _, edge := range edges {
		childID := edge.ChildAccountID
		if activePath[childID] {
			// Cycle detected targeting child node: prune circular branch
			continue
		}

		nextVotingWeight := currentVotingWeight.Mul(edge.VotingControlPct)
		nextEconomicWeight := currentEconomicWeight.Mul(edge.EconomicSharePct)

		traverseHierarchy(childID, adjMap, activePath, nextVotingWeight, nextEconomicWeight, issuerVotingMap, issuerTakeoverMap)
	}
}
