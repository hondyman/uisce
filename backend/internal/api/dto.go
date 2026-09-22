package api

type generateTermItem struct {
	Name      string   `json:"name"`
	ColumnIDs []string `json:"column_ids"`
}

type generateTermResult struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ReusedExisting     bool   `json:"reused_existing"`
	BusinessTermID     string `json:"business_term_id"`
	BusinessTermName   string `json:"business_term_name"`
	BusinessTermReused bool   `json:"business_term_reused"`
	DefinitionSource   string `json:"definition_source"`
	ColumnsLinked      int    `json:"columns_linked"`
	ColumnsTotal       int    `json:"columns_total"`
	Error              string `json:"error,omitempty"`
}

type rawGenerateTermsRequest struct {
	Name      string             `json:"name"`
	ColumnIDs []string           `json:"column_ids"`
	Items     []generateTermItem `json:"items"`
}

func (r *rawGenerateTermsRequest) Normalize() ([]generateTermItem, error) {
	if len(r.Items) > 0 {
		return r.Items, nil
	}
	if len(r.ColumnIDs) == 0 {
		return nil, ErrColumnIDsRequired
	}
	return []generateTermItem{{Name: r.Name, ColumnIDs: r.ColumnIDs}}, nil
}

type generateTermsResponse struct {
	Success        bool                 `json:"success"`
	TotalRequested int                  `json:"total_requested,omitempty"`
	CreatedTerms   int                  `json:"created_terms,omitempty"`
	ReusedTerms    int                  `json:"reused_terms,omitempty"`
	ColumnsLinked  int                  `json:"columns_linked,omitempty"`
	Results        []generateTermResult `json:"results,omitempty"`
	JobID          string               `json:"job_id,omitempty"`
}

var ErrColumnIDsRequired = errColumnIDsRequired{}

type errColumnIDsRequired struct{}

func (e errColumnIDsRequired) Error() string { return "column_ids or items is required" }

// PreviewResult is returned by POST /api/glossary/preview-semantic-terms.
// Source is one of:
//   - "pascal": raw pascal-case of column name, no abbreviation matched
//   - "abbrev_map": abbreviation table resolved at least one token
//   - "addr_line_context": address_line_N contextual rule applied (table context used)
//   - "bare_generic": bare generic word qualified by table name (LLM may refine on create)
type PreviewResult struct {
	ColumnID     string `json:"column_id"`
	SemanticName string `json:"semantic_name"`
	BusinessName string `json:"business_name"`
	Source       string `json:"source"`
}

type previewSemanticTermsRequest struct {
	ColumnIDs []string `json:"column_ids"`
}

type previewSemanticTermsResponse struct {
	Suggestions []PreviewResult `json:"suggestions"`
}

type rejectRequest struct {
	ColumnID       string `json:"column_id"`
	RejectedName   string `json:"rejected_name"`
}

type unrejectRequest struct {
	ColumnID       string `json:"column_id"`
	RejectedName   string `json:"rejected_name"`
}
