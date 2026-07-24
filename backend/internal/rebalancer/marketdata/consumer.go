package marketdata

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// MarketDataConsumer simulates consuming market data from an event bus (e.g., NATS/Kafka)
type MarketDataConsumer struct {
	db *sqlx.DB
}

func NewMarketDataConsumer(db *sqlx.DB) *MarketDataConsumer {
	return &MarketDataConsumer{db: db}
}

// Start begins consuming messages (simulated loop)
func (c *MarketDataConsumer) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.processSimulatedMessage()
			}
		}
	}()
}

func (c *MarketDataConsumer) processSimulatedMessage() {
	// Simulate receiving a price update
	// In reality, this would decode a Protobuf/JSON message from NATS
	
	// Example: AAPL price update
	query := `
		INSERT INTO market_data (ticker, date, close_price, currency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (ticker, date) DO UPDATE 
		SET close_price = EXCLUDED.close_price
	`
	
	// Randomize price slightly for demo
	price := 175.0 + (float64(time.Now().Unix()%100) / 100.0)
	
	_, err := c.db.Exec(query, "AAPL", time.Now(), price, "USD")
	if err != nil {
		fmt.Printf("Error processing market data: %v\n", err)
	} else {
		// fmt.Printf("Updated AAPL price: %.2f\n", price)
	}
}
