package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ABACCheck performs ABAC authorization check for UMA rebalance
func ABACCheck(ctx context.Context, action, resourceType, resourceID string) (bool, error) {
	// Call ABAC governance service for permission check
	abacURL := os.Getenv("ABAC_SERVICE_URL")
	if abacURL == "" {
		abacURL = "http://localhost:8083" // Default governance service URL
	}

	request := map[string]interface{}{
		"action":       action,
		"resourceType": resourceType,
		"resourceID":   resourceID,
		"tenantID":     getTenantFromContext(ctx),
		"userID":       getUserFromContext(ctx),
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(abacURL+"/api/abac/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		// If ABAC service is unavailable, log and allow (fail open for now)
		fmt.Printf("ABAC service unavailable, granting permission: %v\n", err)
		return true, nil
	}
	defer resp.Body.Close()

	var result struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode ABAC response: %w", err)
	}

	fmt.Printf("ABAC Check: action=%s, resource=%s:%s, allowed=%v, reason=%s\n",
		action, resourceType, resourceID, result.Allowed, result.Reason)

	return result.Allowed, nil
}

// ExecuteTrades executes the trades based on the harvest plan
func ExecuteTrades(ctx context.Context, harvest map[string]any) error {
	// Extract trade details from harvest plan
	trades, ok := harvest["trades"].([]interface{})
	if !ok {
		return fmt.Errorf("no trades found in harvest plan")
	}

	fmt.Printf("Executing %d trades for UMA rebalance\n", len(trades))

	for i, trade := range trades {
		tradeMap, ok := trade.(map[string]interface{})
		if !ok {
			continue
		}

		// Execute trade via trading API
		err := executeSingleTrade(ctx, tradeMap)
		if err != nil {
			return fmt.Errorf("failed to execute trade %d: %w", i, err)
		}

		fmt.Printf("  ✓ Trade %d executed: %v\n", i+1, tradeMap)
	}

	return nil
}

// executeSingleTrade executes a single trade
func executeSingleTrade(ctx context.Context, trade map[string]interface{}) error {
	tradingURL := os.Getenv("TRADING_API_URL")
	if tradingURL == "" {
		// If no trading API configured, just log the trade
		fmt.Printf("    [SIMULATED] Trade: %+v\n", trade)
		return nil
	}

	body, _ := json.Marshal(trade)
	resp, err := http.Post(tradingURL+"/api/trades", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("trading API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("trade execution failed with status %d", resp.StatusCode)
	}

	return nil
}

var dbPool *pgxpool.Pool

func initDB() error {
	if dbPool != nil {
		return nil
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/portfolio"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("failed to create db pool: %w", err)
	}
	dbPool = pool
	return nil
}

func getPool() *pgxpool.Pool {
	if dbPool == nil {
		initDB()
	}
	return dbPool
}

// HasuraUpdate updates the database with rebalance results
func HasuraUpdate(ctx context.Context, update map[string]any) error {
	pool := getPool()
	if pool == nil {
		return fmt.Errorf("database not initialized")
	}

	updateType, _ := update["type"].(string)
	entityID, _ := update["entityID"].(string)

	switch updateType {
	case "rebalance_complete":
		resultsJSON, _ := json.Marshal(update["results"])
		_, err := pool.Exec(ctx, `
			UPDATE rebalance_executions
			SET status = 'completed', results = $2, completed_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, entityID, resultsJSON)
		if err != nil {
			return fmt.Errorf("failed to update rebalance execution: %w", err)
		}

	case "portfolio_update":
		holdingsJSON, _ := json.Marshal(update["holdings"])
		_, err := pool.Exec(ctx, `
			UPDATE portfolios
			SET holdings = $2, updated_at = NOW()
			WHERE id = $1
		`, entityID, holdingsJSON)
		if err != nil {
			return fmt.Errorf("failed to update portfolio: %w", err)
		}

	default:
		return fmt.Errorf("unknown update type: %s", updateType)
	}

	fmt.Printf("Database updated successfully: type=%s, entityID=%s\n", updateType, entityID)
	return nil
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// Helper functions to extract context values
func getTenantFromContext(ctx context.Context) string {
	if tenantID := ctx.Value("tenantID"); tenantID != nil {
		if tid, ok := tenantID.(string); ok {
			return tid
		}
	}
	return "default"
}

func getUserFromContext(ctx context.Context) string {
	if userID := ctx.Value("userID"); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "system"
}
