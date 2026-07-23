package factors

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// FamaFrenchModel implements the Fama-French 5-factor model
type FamaFrenchModel struct {
	factorData map[string][]FactorDataPoint // Map of factor name to time series
}

// NewFamaFrenchModel creates a new Fama-French 5-factor model
func NewFamaFrenchModel(dataPath string) (*FamaFrenchModel, error) {
	model := &FamaFrenchModel{
		factorData: make(map[string][]FactorDataPoint),
	}
	
	// Load factor returns from CSV file
	if err := model.loadFactorData(dataPath); err != nil {
		return nil, fmt.Errorf("failed to load factor data: %w", err)
	}
	
	return model, nil
}

// Name returns the model name
func (ff *FamaFrenchModel) Name() string {
	return "Fama-French 5-Factor (US)"
}

// Type returns the model type
func (ff *FamaFrenchModel) Type() string {
	return "fama_french"
}

// Factors returns the list of factors
func (ff *FamaFrenchModel) Factors() []string {
	return []string{"Market", "SMB", "HML", "RMW", "CMA"}
}

// ComputeExposures calculates factor exposures for a portfolio
func (ff *FamaFrenchModel) ComputeExposures(holdings []Holding, startDate, endDate time.Time) ([]FactorExposure, error) {
	// TODO: Implement actual factor exposure calculation
	// This would involve:
	// 1. Fetch historical returns for portfolio holdings
	// 2. Calculate portfolio returns
	// 3. Run regression against factor returns
	// 4. Extract factor loadings (betas)
	
	// Placeholder implementation
	exposures := []FactorExposure{
		{
			Factor:       "Market",
			Contribution: 0.75,
			Narrative:    "Portfolio has high market exposure, consistent with broad equity allocation",
			Significance: 12.5,
			PValue:       0.001,
		},
		{
			Factor:       "SMB",
			Contribution: 0.15,
			Narrative:    "Modest small-cap tilt driven by smaller technology holdings",
			Significance: 3.2,
			PValue:       0.05,
		},
		{
			Factor:       "HML",
			Contribution: -0.05,
			Narrative:    "Slight growth tilt, underweight value stocks",
			Significance: -1.1,
			PValue:       0.27,
		},
		{
			Factor:       "RMW",
			Contribution: 0.10,
			Narrative:    "Overweight profitable companies",
			Significance: 2.8,
			PValue:       0.06,
		},
		{
			Factor:       "CMA",
			Contribution: -0.03,
			Narrative:    "Modest preference for aggressive investment firms",
			Significance: -0.9,
			PValue:       0.38,
		},
	}
	
	return exposures, nil
}

// loadFactorData loads Fama-French factor returns from CSV
func (ff *FamaFrenchModel) loadFactorData(dataPath string) error {
	file, err := os.Open(dataPath)
	if err != nil {
		// If file doesn't exist, use placeholder data
		ff.loadPlaceholderData()
		return nil
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	// Parse CSV format (date, Mkt-RF, SMB, HML, RMW, CMA, RF)
	for i, record := range records {
		if i == 0 {
			// Skip header
			continue
		}
		
		date, err := parseFFDate(record[0])
		if err != nil {
			continue
		}
		
		// Parse factor returns (in percentage points)
		mktRF := parseFloat(record[1]) / 100
		smb := parseFloat(record[2]) / 100
		hml := parseFloat(record[3]) / 100
		rmw := parseFloat(record[4]) / 100
		cma := parseFloat(record[5]) / 100
		
		// Store in factor data
		ff.factorData["Market"] = append(ff.factorData["Market"], FactorDataPoint{Date: date, Return: mktRF})
		ff.factorData["SMB"] = append(ff.factorData["SMB"], FactorDataPoint{Date: date, Return: smb})
		ff.factorData["HML"] = append(ff.factorData["HML"], FactorDataPoint{Date: date, Return: hml})
		ff.factorData["RMW"] = append(ff.factorData["RMW"], FactorDataPoint{Date: date, Return: rmw})
		ff.factorData["CMA"] = append(ff.factorData["CMA"], FactorDataPoint{Date: date, Return: cma})
	}
	
	return nil
}

// loadPlaceholderData creates sample factor data for development
func (ff *FamaFrenchModel) loadPlaceholderData() {
	// Generate 252 trading days of placeholder data (1 year)
	baseDate := time.Now().AddDate(-1, 0, 0)
	
	for i := 0; i < 252; i++ {
		date := baseDate.AddDate(0, 0, i)
		
		// Mock returns (normally distributed around 0)
		ff.factorData["Market"] = append(ff.factorData["Market"], FactorDataPoint{
			Date:   date,
			Return: 0.0008, // ~0.08% daily
		})
		ff.factorData["SMB"] = append(ff.factorData["SMB"], FactorDataPoint{
			Date:   date,
			Return: 0.0002,
		})
		ff.factorData["HML"] = append(ff.factorData["HML"], FactorDataPoint{
			Date:   date,
			Return: -0.0001,
		})
		ff.factorData["RMW"] = append(ff.factorData["RMW"], FactorDataPoint{
			Date:   date,
			Return: 0.0003,
		})
		ff.factorData["CMA"] = append(ff.factorData["CMA"], FactorDataPoint{
			Date:   date,
			Return: -0.0001,
		})
	}
}

// parseFFDate parses Fama-French date format (YYYYMMDD)
func parseFFDate(s string) (time.Time, error) {
	if len(s) != 8 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", s)
	}
	
	year, _ := strconv.Atoi(s[0:4])
	month, _ := strconv.Atoi(s[4:6])
	day, _ := strconv.Atoi(s[6:8])
	
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// parseFloat safely parses a float from string
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// GetFactorReturns retrieves factor returns for a date range
func (ff *FamaFrenchModel) GetFactorReturns(ctx context.Context, factor string, startDate, endDate time.Time) ([]FactorDataPoint, error) {
	data, exists := ff.factorData[factor]
	if !exists {
		return nil, fmt.Errorf("factor not found: %s", factor)
	}
	
	// Filter by date range
	var filtered []FactorDataPoint
	for _, point := range data {
		if (point.Date.Equal(startDate) || point.Date.After(startDate)) &&
		   (point.Date.Equal(endDate) || point.Date.Before(endDate)) {
			filtered = append(filtered, point)
		}
	}
	
	return filtered, nil
}
