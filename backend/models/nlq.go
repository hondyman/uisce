package models

// NLQTranslateRequest is the request for the natural language query translation endpoint.
type NLQTranslateRequest struct {
	Text         string `json:"text" binding:"required"`
	DatasourceID string `json:"datasource_id" binding:"required"`
	UserID       string `json:"user_id" binding:"required"`
	ViewName     string `json:"view_name,omitempty"`
}

// NLQTranslateResponse is the response from the natural language query translation endpoint.
type NLQTranslateResponse struct {
	ViewName    string        `json:"view_name"`
	Query       SemanticQuery `json:"query"`
	Explanation string        `json:"explanation"`
}
