package surveillance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AssetPosition represents an asset holding in a portfolio
type AssetPosition struct {
	PositionID     uuid.UUID
	AccountID      uuid.UUID
	SecurityID     uuid.UUID
	IssuerID       uuid.UUID
	AssetClass     string // "EQUITY", "ETF", "OPTION", "SWAP"
	MarketValue    decimal.Decimal
	Delta          decimal.Decimal // For options/derivatives, 1.0 for cash equity
	UnderlyingID   *uuid.UUID      // For derivatives
}

// FundConstituent represents a lookthrough constituent holding inside a fund/ETF
type FundConstituent struct {
	FundSecurityID uuid.UUID
	ConstituentID  uuid.UUID
	IssuerID       uuid.UUID
	Weight         decimal.Decimal // Weight as decimal e.g. 0.075 for 7.5%
}

// IssuerExposureSummary aggregates total effective single-issuer exposure across all instruments
type IssuerExposureSummary struct {
	IssuerID             uuid.UUID
	DirectExposure       decimal.Decimal
	FundLookthroughValue decimal.Decimal
	DerivativeExposure   decimal.Decimal
	TotalExposure        decimal.Decimal
	PercentOfNAV         decimal.Decimal
}

// MultiAssetLookthroughAggregator traverses complex portfolio structures for ultimate single-issuer risk
type MultiAssetLookthroughAggregator struct {
	fundConstituents map[uuid.UUID][]FundConstituent // FundSecurityID -> constituents
}

func NewMultiAssetLookthroughAggregator(constituents []FundConstituent) *MultiAssetLookthroughAggregator {
	m := make(map[uuid.UUID][]FundConstituent)
	for _, c := range constituents {
		m[c.FundSecurityID] = append(m[c.FundSecurityID], c)
	}
	return &MultiAssetLookthroughAggregator{
		fundConstituents: m,
	}
}

// AggregateIssuerExposures aggregates direct, fund lookthrough, and derivative exposures for an account NAV
func (a *MultiAssetLookthroughAggregator) AggregateIssuerExposures(ctx context.Context, accountNAV decimal.Decimal, positions []AssetPosition) (map[uuid.UUID]*IssuerExposureSummary, error) {
	if accountNAV.IsZero() {
		return nil, fmt.Errorf("account NAV cannot be zero")
	}

	summaries := make(map[uuid.UUID]*IssuerExposureSummary)

	getSummary := func(issuerID uuid.UUID) *IssuerExposureSummary {
		if s, ok := summaries[issuerID]; ok {
			return s
		}
		s := &IssuerExposureSummary{
			IssuerID:             issuerID,
			DirectExposure:       decimal.Zero,
			FundLookthroughValue: decimal.Zero,
			DerivativeExposure:   decimal.Zero,
			TotalExposure:        decimal.Zero,
			PercentOfNAV:         decimal.Zero,
		}
		summaries[issuerID] = s
		return s
	}

	for _, pos := range positions {
		switch pos.AssetClass {
		case "EQUITY", "BOND":
			s := getSummary(pos.IssuerID)
			s.DirectExposure = s.DirectExposure.Add(pos.MarketValue)

		case "ETF", "MUTUAL_FUND":
			constituents, ok := a.fundConstituents[pos.SecurityID]
			if ok && len(constituents) > 0 {
				for _, c := range constituents {
					lookthroughAmt := pos.MarketValue.Mul(c.Weight)
					s := getSummary(c.IssuerID)
					s.FundLookthroughValue = s.FundLookthroughValue.Add(lookthroughAmt)
				}
			} else {
				// No constituent breakdown available, treat fund as its own issuer
				s := getSummary(pos.IssuerID)
				s.DirectExposure = s.DirectExposure.Add(pos.MarketValue)
			}

		case "OPTION", "SWAP", "DERIVATIVE":
			delta := pos.Delta
			if delta.IsZero() {
				delta = decimal.NewFromInt(1)
			}
			effectiveExposure := pos.MarketValue.Mul(delta)
			s := getSummary(pos.IssuerID)
			s.DerivativeExposure = s.DerivativeExposure.Add(effectiveExposure)
		}
	}

	// Compute totals and percentage of account NAV
	for _, s := range summaries {
		s.TotalExposure = s.DirectExposure.Add(s.FundLookthroughValue).Add(s.DerivativeExposure)
		s.PercentOfNAV = s.TotalExposure.Div(accountNAV).Mul(decimal.NewFromInt(100))
	}

	return summaries, nil
}
