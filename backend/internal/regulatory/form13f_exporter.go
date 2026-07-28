package regulatory

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// ============================================================
// Form 13F (SEC Institutional Holdings)
// ============================================================

// SEC Form 13F XML Schema Structs
type Form13FReport struct {
	XMLName          xml.Name      `xml:"edgarSubmission"`
	SubmissionType   string        `xml:"submissionType"`
	CIK              string        `xml:"cik"`
	PeriodOfReport   string        `xml:"periodOfReport"`
	InformationTable []HoldingInfo `xml:"informationTable>infoTable"`
}

type HoldingInfo struct {
	NameOfIssuer         string `xml:"nameOfIssuer"`
	TitleOfClass         string `xml:"titleOfClass"`
	Cusip                string `xml:"cusip"`
	Value                int64  `xml:"value"` // In thousands USD
	ShrsOrPrnAmt         Shrs   `xml:"shrsOrPrnAmt"`
	InvestmentDiscretion string `xml:"investmentDiscretion"` // SOLE, SHARED
	PutCall              string `xml:"putCall,omitempty"`
	OtherManager         string `xml:"otherManager,omitempty"`
	VotingAuthority      VotAuth `xml:"votingAuthority"`
}

type Shrs struct {
	SSHPrnAmt  int64  `xml:"sshPrnAmt"`
	SSHPrnType string `xml:"sshPrnType"` // SH or PRN
}

type VotAuth struct {
	Sole    int64 `xml:"Sole"`
	Shared  int64 `xml:"Shared"`
	None    int64 `xml:"None"`
}

// ============================================================
// Form PF (SEC Private Fund Reporting)
// ============================================================

type FormPFReport struct {
	ReportingPeriod   string          `json:"reporting_period"`
	FilingManagerName string          `json:"filing_manager_name"`
	CRD               string          `json:"crd_number"`
	AggregateAUM      float64         `json:"aggregate_aum_millions"`
	Funds             []FormPFSection1 `json:"funds"`
	GeneratedAt       time.Time       `json:"generated_at"`
}

type FormPFSection1 struct {
	FundID          string  `json:"fund_id"`
	FundName        string  `json:"fund_name"`
	FundType        string  `json:"fund_type"`    // Liquidity, Hedge, PE, RE, VC
	AUM             float64 `json:"aum_millions"`
	NAVPerShare     float64 `json:"nav_per_share"`
	Leverage        float64 `json:"gross_leverage_ratio"`
	LiquidAssetsPct float64 `json:"liquid_assets_pct"`
	Strategy        string  `json:"strategy"`
	Currency        string  `json:"currency"`
	Domicile        string  `json:"domicile"`
}

// ============================================================
// Basel III Capital Adequacy
// ============================================================

type Basel3Report struct {
	InstitutionName    string         `json:"institution_name"`
	ReportingDate      time.Time      `json:"reporting_date"`
	Tier1Capital       float64        `json:"tier1_capital_millions"`
	Tier2Capital       float64        `json:"tier2_capital_millions"`
	TotalCapital       float64        `json:"total_capital_millions"`
	RWA                float64        `json:"risk_weighted_assets_millions"`
	CET1Ratio          float64        `json:"cet1_ratio_pct"`
	Tier1Ratio         float64        `json:"tier1_ratio_pct"`
	TotalCapitalRatio  float64        `json:"total_capital_ratio_pct"`
	LCR                float64        `json:"liquidity_coverage_ratio_pct"`
	NSFR               float64        `json:"net_stable_funding_ratio_pct"`
	LeverageRatio      float64        `json:"leverage_ratio_pct"`
	Breaches           []string       `json:"regulatory_breaches"`
	Status             string         `json:"status"` // COMPLIANT, BREACH, WARNING
}

// ============================================================
// Regulatory Service
// ============================================================

type RegulatoryService struct {
	db *sqlx.DB
}

func NewRegulatoryService(db *sqlx.DB) *RegulatoryService {
	return &RegulatoryService{db: db}
}

// GenerateForm13FXML compiles holdings into SEC-compliant Form 13F XML
func GenerateForm13FXML(ctx context.Context, cik string, periodOfReport string, holdings []HoldingInfo) ([]byte, error) {
	if cik == "" {
		cik = "0001234567"
	}
	if periodOfReport == "" {
		now := time.Now()
		// Last quarter end
		periodOfReport = fmt.Sprintf("%d-%02d-%02d", now.Year(), ((now.Month()-1)/3)*3+1, 1)
	}
	if len(holdings) == 0 {
		holdings = []HoldingInfo{
			{
				NameOfIssuer: "APPLE INC", TitleOfClass: "COM", Cusip: "037833100",
				Value: 220000, ShrsOrPrnAmt: Shrs{SSHPrnAmt: 1000, SSHPrnType: "SH"},
				InvestmentDiscretion: "SOLE", VotingAuthority: VotAuth{Sole: 1000},
			},
			{
				NameOfIssuer: "MICROSOFT CORP", TitleOfClass: "COM", Cusip: "594918104",
				Value: 450000, ShrsOrPrnAmt: Shrs{SSHPrnAmt: 2000, SSHPrnType: "SH"},
				InvestmentDiscretion: "SOLE", VotingAuthority: VotAuth{Sole: 2000},
			},
			{
				NameOfIssuer: "AMAZON.COM INC", TitleOfClass: "COM", Cusip: "023135106",
				Value: 185000, ShrsOrPrnAmt: Shrs{SSHPrnAmt: 500, SSHPrnType: "SH"},
				InvestmentDiscretion: "SHARED", VotingAuthority: VotAuth{Shared: 500},
			},
		}
	}

	report := Form13FReport{
		SubmissionType:   "13F-HR",
		CIK:              cik,
		PeriodOfReport:   periodOfReport,
		InformationTable: holdings,
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(report); err != nil {
		return nil, fmt.Errorf("failed to marshal Form 13F XML: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateFormPF compiles private fund data into SEC Form PF JSON report
func GenerateFormPF(ctx context.Context, managerName, crd string, funds []FormPFSection1) (*FormPFReport, error) {
	var totalAUM float64
	for _, f := range funds {
		totalAUM += f.AUM
	}
	if len(funds) == 0 {
		funds = []FormPFSection1{
			{FundID: "FUND001", FundName: "Alpha Growth Fund", FundType: "Hedge",
				AUM: 1250.0, NAVPerShare: 1.0842, Leverage: 1.5, LiquidAssetsPct: 78.5,
				Strategy: "Long/Short Equity", Currency: "USD", Domicile: "Cayman Islands"},
			{FundID: "FUND002", FundName: "Fixed Income Opportunity Fund", FundType: "Liquidity",
				AUM: 450.0, NAVPerShare: 1.0010, Leverage: 1.1, LiquidAssetsPct: 95.2,
				Strategy: "Investment Grade Credit", Currency: "USD", Domicile: "Delaware"},
		}
		for _, f := range funds {
			totalAUM += f.AUM
		}
	}

	return &FormPFReport{
		ReportingPeriod:   time.Now().Format("2006-Q"),
		FilingManagerName: managerName,
		CRD:               crd,
		AggregateAUM:      totalAUM,
		Funds:             funds,
		GeneratedAt:       time.Now(),
	}, nil
}

// CalculateBasel3Ratios computes capital adequacy ratios per Basel III framework
func CalculateBasel3Ratios(institutionName string, t1Capital, t2Capital, rwa, hqla, netOutflows, availFunding, reqFunding float64) *Basel3Report {
	totalCapital := t1Capital + t2Capital
	var breaches []string
	status := "COMPLIANT"

	cet1 := 0.0
	if rwa > 0 {
		cet1 = (t1Capital / rwa) * 100
	}
	tier1Ratio := cet1
	totalCapRatio := 0.0
	if rwa > 0 {
		totalCapRatio = (totalCapital / rwa) * 100
	}
	lcr := 0.0
	if netOutflows > 0 {
		lcr = (hqla / netOutflows) * 100
	}
	nsfr := 0.0
	if reqFunding > 0 {
		nsfr = (availFunding / reqFunding) * 100
	}
	leverageRatio := 0.0
	if rwa > 0 {
		leverageRatio = (t1Capital / (rwa * 1.25)) * 100 // simplified exposure measure
	}

	// Check regulatory minima
	if cet1 < 4.5 {
		breaches = append(breaches, fmt.Sprintf("CET1 Ratio %.2f%% below minimum 4.5%%", cet1))
		status = "BREACH"
	} else if cet1 < 7.0 {
		breaches = append(breaches, fmt.Sprintf("CET1 Ratio %.2f%% in conservation buffer zone", cet1))
		if status == "COMPLIANT" {
			status = "WARNING"
		}
	}
	if tier1Ratio < 6.0 {
		breaches = append(breaches, fmt.Sprintf("Tier 1 Ratio %.2f%% below minimum 6.0%%", tier1Ratio))
		status = "BREACH"
	}
	if totalCapRatio < 8.0 {
		breaches = append(breaches, fmt.Sprintf("Total Capital Ratio %.2f%% below minimum 8.0%%", totalCapRatio))
		status = "BREACH"
	}
	if lcr < 100 && netOutflows > 0 {
		breaches = append(breaches, fmt.Sprintf("LCR %.2f%% below minimum 100%%", lcr))
		status = "BREACH"
	}
	if nsfr < 100 && reqFunding > 0 {
		breaches = append(breaches, fmt.Sprintf("NSFR %.2f%% below minimum 100%%", nsfr))
		status = "BREACH"
	}

	return &Basel3Report{
		InstitutionName:   institutionName,
		ReportingDate:     time.Now(),
		Tier1Capital:      t1Capital,
		Tier2Capital:      t2Capital,
		TotalCapital:      totalCapital,
		RWA:               rwa,
		CET1Ratio:         cet1,
		Tier1Ratio:        tier1Ratio,
		TotalCapitalRatio: totalCapRatio,
		LCR:               lcr,
		NSFR:              nsfr,
		LeverageRatio:     leverageRatio,
		Breaches:          breaches,
		Status:            status,
	}
}

// ============================================================
// HTTP Handlers
// ============================================================

func (s *RegulatoryService) GenerateForm13FHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CIK            string        `json:"cik"`
		PeriodOfReport string        `json:"period_of_report"`
		Holdings       []HoldingInfo `json:"holdings"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	xmlData, err := GenerateForm13FXML(r.Context(), body.CIK, body.PeriodOfReport, body.Holdings)
	if err != nil {
		http.Error(w, fmt.Sprintf("Form 13F generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="sec_form_13f_filing.xml"`)
	w.Write(xmlData)
}

func (s *RegulatoryService) GenerateFormPFHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	_ = tenantID

	var body struct {
		ManagerName string          `json:"manager_name"`
		CRD         string          `json:"crd"`
		Funds       []FormPFSection1 `json:"funds"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	report, err := GenerateFormPF(r.Context(), body.ManagerName, body.CRD, body.Funds)
	if err != nil {
		http.Error(w, fmt.Sprintf("Form PF generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="form_pf_report.json"`)
	json.NewEncoder(w).Encode(report)
}

func (s *RegulatoryService) GenerateBasel3Handler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstitutionName string  `json:"institution_name"`
		Tier1Capital    float64 `json:"tier1_capital_millions"`
		Tier2Capital    float64 `json:"tier2_capital_millions"`
		RWA             float64 `json:"rwa_millions"`
		HQLA            float64 `json:"hqla_millions"`
		NetOutflows     float64 `json:"net_outflows_millions"`
		AvailFunding    float64 `json:"available_stable_funding_millions"`
		ReqFunding      float64 `json:"required_stable_funding_millions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Use demo defaults if no body
		body.InstitutionName = "Uisce Financial Institution"
		body.Tier1Capital = 850.0
		body.Tier2Capital = 150.0
		body.RWA = 8500.0
		body.HQLA = 1200.0
		body.NetOutflows = 1000.0
		body.AvailFunding = 5500.0
		body.ReqFunding = 5000.0
	}

	report := CalculateBasel3Ratios(
		body.InstitutionName,
		body.Tier1Capital, body.Tier2Capital, body.RWA,
		body.HQLA, body.NetOutflows, body.AvailFunding, body.ReqFunding,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GenerateRegulatoryFormHandler is a unified endpoint that routes to the correct form generator
func (s *RegulatoryService) GenerateRegulatoryFormHandler(w http.ResponseWriter, r *http.Request) {
	formType := r.URL.Query().Get("form_type")
	switch formType {
	case "13F", "SEC_13F":
		s.GenerateForm13FHandler(w, r)
	case "PF", "FORM_PF":
		s.GenerateFormPFHandler(w, r)
	case "BASEL3", "BASEL_III":
		s.GenerateBasel3Handler(w, r)
	default:
		http.Error(w, fmt.Sprintf("unknown form_type: %s. Valid: 13F, PF, BASEL3", formType), http.StatusBadRequest)
	}
}
