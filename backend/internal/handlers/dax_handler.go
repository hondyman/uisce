package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// DAXHandler handles DAX function API requests
type DAXHandler struct {
	daxEngine *services.DAXEngine
}

// NewDAXHandler creates a new DAX handler
func NewDAXHandler() *DAXHandler {
	return &DAXHandler{
		daxEngine: services.NewDAXEngine(),
	}
}

// HandleExecuteFunction executes a DAX function
func (h *DAXHandler) HandleExecuteFunction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FunctionName string        `json:"function_name"`
		Arguments    []interface{} `json:"arguments"`
		Context      struct {
			TableName  string                 `json:"table_name,omitempty"`
			ColumnName string                 `json:"column_name,omitempty"`
			Filters    map[string]interface{} `json:"filters,omitempty"`
			DateColumn string                 `json:"date_column,omitempty"`
		} `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.FunctionName == "" {
		http.Error(w, "function_name is required", http.StatusBadRequest)
		return
	}

	context := &services.DAXContext{
		TableName:  req.Context.TableName,
		ColumnName: req.Context.ColumnName,
		Filters:    req.Context.Filters,
		DateColumn: req.Context.DateColumn,
	}

	result, err := h.daxEngine.ExecuteFunction(req.FunctionName, req.Arguments, context)
	if err != nil {
		http.Error(w, "Function execution failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"function": req.FunctionName,
		"result":   result,
	})
}

// HandleListFunctions returns all available DAX functions
func (h *DAXHandler) HandleListFunctions(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	var functions map[string]*services.DAXFunction
	if category != "" {
		categories := h.daxEngine.ListFunctionsByCategory()
		if funcs, exists := categories[category]; exists {
			functions = make(map[string]*services.DAXFunction)
			for _, f := range funcs {
				functions[f.Name] = f
			}
		} else {
			functions = make(map[string]*services.DAXFunction)
		}
	} else {
		functions = h.daxEngine.ListFunctions()
	}

	// Convert to JSON-friendly format
	functionList := make([]map[string]interface{}, 0, len(functions))
	for _, f := range functions {
		functionList = append(functionList, map[string]interface{}{
			"name":        f.Name,
			"category":    f.Category,
			"description": f.Description,
			"min_args":    f.MinArgs,
			"max_args":    f.MaxArgs,
			"return_type": f.ReturnType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"functions": functionList,
		"count":     len(functionList),
	})
}

// HandleGetFunctionInfo returns information about a specific DAX function
func (h *DAXHandler) HandleGetFunctionInfo(w http.ResponseWriter, r *http.Request) {
	functionName := chi.URLParam(r, "functionName")
	if functionName == "" {
		http.Error(w, "function name is required", http.StatusBadRequest)
		return
	}

	function, exists := h.daxEngine.GetFunctionInfo(strings.ToUpper(functionName))
	if !exists {
		http.Error(w, "Function not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        function.Name,
		"category":    function.Category,
		"description": function.Description,
		"min_args":    function.MinArgs,
		"max_args":    function.MaxArgs,
		"return_type": function.ReturnType,
	})
}

// HandleListCategories returns all function categories
func (h *DAXHandler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	categories := h.daxEngine.ListFunctionsByCategory()

	categoryList := make([]string, 0, len(categories))
	for category := range categories {
		categoryList = append(categoryList, category)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categoryList,
		"count":      len(categoryList),
	})
}

// RegisterRoutes registers the DAX routes
func (h *DAXHandler) RegisterRoutes(r chi.Router) {
	r.Route("/dax", func(r chi.Router) {
		r.Post("/execute", h.HandleExecuteFunction)
		r.Get("/functions", h.HandleListFunctions)
		r.Get("/functions/{functionName}", h.HandleGetFunctionInfo)
		r.Get("/categories", h.HandleListCategories)
	})
}
