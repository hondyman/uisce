package templates

import (
	"context"
)

type RegulatoryTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"` // KYC, AML, MiFID
	Description string `json:"description"`
	PageConfig  string `json:"page_config"` // JSON string of the page structure
}

type TemplateLoader struct{}

func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{}
}

func (l *TemplateLoader) ListTemplates(ctx context.Context) ([]RegulatoryTemplate, error) {
	return []RegulatoryTemplate{
		{
			ID:          "kyc_verification_v1",
			Name:        "KYC Identity Verification",
			Category:    "KYC",
			Description: "Standard workflow for identity proofing and risk scoring.",
			PageConfig:  `{"layout": "stepper", "components": ["IdentityForm", "DocUpload", "RiskScore"]}`,
		},
		{
			ID:          "aml_transaction_monitor",
			Name:        "AML Transaction Monitor",
			Category:    "AML",
			Description: "Dashboard for reviewing flagged transactions.",
			PageConfig:  `{"layout": "dashboard", "components": ["AlertList", "TxDetail", "BeneficiaryGraph"]}`,
		},
	}, nil
}
