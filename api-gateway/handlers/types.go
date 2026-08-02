package handlers

type BusinessTermSearchRequest struct {
	Query    string `json:"query"`
	TenantID string `json:"tenant_id,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type BusinessTermValidationRequest struct {
	TermID   string                 `json:"term_id,omitempty"`
	Name     string                 `json:"name"`
	Category string                 `json:"category"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
