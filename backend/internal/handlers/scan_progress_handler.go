package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
)

// HandleScanStream handles SSE streaming of scan progress
func (h *CatalogScanHandler) HandleScanStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers BEFORE any writes
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Access-Control-Allow-Origin is handled by CORS middleware usually,
	// but adding it here ensures it works for direct calls too if middleware is skipped
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	// Get datasource ID from query
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		http.Error(w, "datasource_id is required", http.StatusBadRequest)
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		http.Error(w, "invalid datasource_id", http.StatusBadRequest)
		return
	}

	// Use http.ResponseController for flushing (Go 1.20+)
	rc := http.NewResponseController(w)

	// Send initial connection message to force headers to be sent immediately.
	// This ensures the client receives the 'open' event right away.
	fmt.Fprintf(w, ": connected\n\n")
	if err := rc.Flush(); err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to flush initial SSE headers: %v", err)
		return
	}

	// Create progress channel
	progressChan := make(chan models.ScanProgress, 100)

	// Create context that cancels when client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start scan in goroutine, passing progress channel
	go func() {
		defer close(progressChan)
		logging.GetLogger().Sugar().Infof("Starting scan for datasource %s", datasourceID)
		_, scanErr := h.scanService.ScanWithProgress(ctx, &datasourceID, progressChan)
		if scanErr != nil {
			logging.GetLogger().Sugar().Errorf("Scan error for %s: %v", datasourceID, scanErr)
		}
	}()

	// Stream progress to client
	for progress := range progressChan {
		data, err := json.Marshal(progress)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to marshal progress: %v", err)
			continue
		}

		_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		if err != nil {
			// Client disconnected
			logging.GetLogger().Sugar().Infof("Client disconnected from SSE stream for %s", datasourceID)
			return
		}

		if flushErr := rc.Flush(); flushErr != nil {
			logging.GetLogger().Sugar().Warnf("Failed to flush progress: %v", flushErr)
			return
		}
	}
}
