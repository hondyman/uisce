package tlh

import (
	"sort"
)

// TaxLot represents a holding with cost basis
type TaxLot struct {
	ID           string
	Ticker       string
	Quantity     float64
	AcquiredDate string // YYYY-MM-DD
	UnitCost     float64
}

// HIFOAllocator allocates sell quantity to lots with highest cost basis
type HIFOAllocator struct{}

func NewHIFOAllocator() *HIFOAllocator {
	return &HIFOAllocator{}
}

// AllocateSells returns a map of LotID -> Quantity to sell
func (a *HIFOAllocator) AllocateSells(lots []TaxLot, quantityToSell float64) map[string]float64 {
	// 1. Sort lots by UnitCost Descending (Highest In)
	// If costs are equal, use FIFO (AcquiredDate Ascending) as tiebreaker
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].UnitCost != lots[j].UnitCost {
			return lots[i].UnitCost > lots[j].UnitCost
		}
		return lots[i].AcquiredDate < lots[j].AcquiredDate
	})

	allocation := make(map[string]float64)
	remaining := quantityToSell

	for _, lot := range lots {
		if remaining <= 0 {
			break
		}

		sellFromLot := 0.0
		if lot.Quantity >= remaining {
			sellFromLot = remaining
		} else {
			sellFromLot = lot.Quantity
		}

		allocation[lot.ID] = sellFromLot
		remaining -= sellFromLot
	}

	return allocation
}
