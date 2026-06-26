package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/types"
)

// CryptoPricingService handles real-time and historical crypto pricing
type CryptoPricingService struct {
	db           *sql.DB
	hasuraClient HasuraClient
	httpClient   *http.Client
	apiKey       string // CoinGecko/CoinMarketCap API key
}

// NewCryptoPricingService creates a new pricing service
func NewCryptoPricingService(db *sql.DB, apiKey string) *CryptoPricingService {
	return &CryptoPricingService{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey: apiKey,
	}
}

// NewCryptoPricingServiceWithHasura creates a new service with Hasura support
func NewCryptoPricingServiceWithHasura(db *sql.DB, hasuraClient HasuraClient, apiKey string) *CryptoPricingService {
	return &CryptoPricingService{
		db:           db,
		hasuraClient: hasuraClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey: apiKey,
	}
}

// CoinGeckoPrice represents price data from CoinGecko API
type CoinGeckoPrice struct {
	CurrentPrice   float64 `json:"current_price"`
	MarketCap      float64 `json:"market_cap"`
	TotalVolume    float64 `json:"total_volume"`
	PriceChange1h  float64 `json:"price_change_percentage_1h_in_currency"`
	PriceChange24h float64 `json:"price_change_percentage_24h_in_currency"`
	PriceChange7d  float64 `json:"price_change_percentage_7d_in_currency"`
	High24h        float64 `json:"high_24h"`
	Low24h         float64 `json:"low_24h"`
}

// FetchCurrentPrice fetches current price from CoinGecko
func (s *CryptoPricingService) FetchCurrentPrice(ctx context.Context, symbol string) (*types.CryptoPrice, error) {
	// Map common symbols to CoinGecko IDs
	coinID := s.symbolToCoinGeckoID(symbol)

	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&price_change_percentage=1h,24h,7d",
		coinID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.apiKey != "" {
		req.Header.Set("X-CG-Pro-API-Key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API error: %d", resp.StatusCode)
	}

	var results []CoinGeckoPrice
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no price data for symbol: %s", symbol)
	}

	data := results[0]

	price := &types.CryptoPrice{
		AssetSymbol:  symbol,
		PriceUSD:     data.CurrentPrice,
		MarketCapUSD: &data.MarketCap,
		Volume24hUSD: &data.TotalVolume,
		Change1hPct:  &data.PriceChange1h,
		Change24hPct: &data.PriceChange24h,
		Change7dPct:  &data.PriceChange7d,
		High24hUSD:   &data.High24h,
		Low24hUSD:    &data.Low24h,
		Source:       "COINGECKO",
		Timestamp:    time.Now(),
	}

	return price, nil
}

// SavePrice saves a price to the database
func (s *CryptoPricingService) SavePrice(ctx context.Context, price *types.CryptoPrice) error {
	return s.savePriceRecord(ctx, price)
}

// GetLatestPrice retrieves the latest cached price for an asset
func (s *CryptoPricingService) GetLatestPrice(ctx context.Context, symbol string) (*types.CryptoPrice, error) {
	return s.getLatestPriceRecord(ctx, symbol)
}

// GetHistoricalPrices retrieves historical prices for an asset
func (s *CryptoPricingService) GetHistoricalPrices(ctx context.Context, symbol string, from, to time.Time) ([]types.CryptoPrice, error) {
	return s.getHistoricalPricesRecords(ctx, symbol, from, to)
}

// UpdateAllPrices fetches and caches prices for all held assets
func (s *CryptoPricingService) UpdateAllPrices(ctx context.Context) error {
	// Get unique asset symbols from holdings
	symbols, err := s.getActiveAssetSymbols(ctx)
	if err != nil {
		return err
	}

	// Fetch and save prices for each symbol
	for _, symbol := range symbols {
		price, err := s.FetchCurrentPrice(ctx, symbol)
		if err != nil {
			// Log error but continue with other symbols
			fmt.Printf("Error fetching price for %s: %v\n", symbol, err)
			continue
		}

		if err := s.SavePrice(ctx, price); err != nil {
			fmt.Printf("Error saving price for %s: %v\n", symbol, err)
		}

		// Rate limiting: wait 1 second between requests (CoinGecko free tier)
		time.Sleep(time.Second)
	}

	return nil
}

// CalculatePortfolioValue calculates total portfolio value for a client
func (s *CryptoPricingService) CalculatePortfolioValue(ctx context.Context, clientID string) (float64, error) {
	return s.calculatePortfolioValueFromHoldings(ctx, clientID)
}

// symbolToCoinGeckoID maps asset symbols to CoinGecko IDs
func (s *CryptoPricingService) symbolToCoinGeckoID(symbol string) string {
	mapping := map[string]string{
		"BTC":   "bitcoin",
		"ETH":   "ethereum",
		"SOL":   "solana",
		"USDC":  "usd-coin",
		"USDT":  "tether",
		"BNB":   "binancecoin",
		"XRP":   "ripple",
		"ADA":   "cardano",
		"DOGE":  "dogecoin",
		"MATIC": "matic-network",
		"DOT":   "polkadot",
		"AVAX":  "avalanche-2",
		"LINK":  "chainlink",
		"UNI":   "uniswap",
		"AAVE":  "aave",
	}

	if id, ok := mapping[symbol]; ok {
		return id
	}

	// Default: lowercase symbol
	return symbol
}

// FetchBatchPrices fetches prices for multiple assets in one request
func (s *CryptoPricingService) FetchBatchPrices(ctx context.Context, symbols []string) (map[string]*types.CryptoPrice, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// Convert symbols to CoinGecko IDs
	ids := make([]string, len(symbols))
	for i, symbol := range symbols {
		ids[i] = s.symbolToCoinGeckoID(symbol)
	}

	// Build comma-separated ID list
	idsParam := ""
	for i, id := range ids {
		if i > 0 {
			idsParam += ","
		}
		idsParam += id
	}

	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&price_change_percentage=1h,24h,7d",
		idsParam,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.apiKey != "" {
		req.Header.Set("X-CG-Pro-API-Key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []CoinGeckoPrice
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	// Map results back to symbols
	priceMap := make(map[string]*types.CryptoPrice)
	for i, data := range results {
		if i < len(symbols) {
			symbol := symbols[i]
			priceMap[symbol] = &types.CryptoPrice{
				AssetSymbol:  symbol,
				PriceUSD:     data.CurrentPrice,
				MarketCapUSD: &data.MarketCap,
				Volume24hUSD: &data.TotalVolume,
				Change1hPct:  &data.PriceChange1h,
				Change24hPct: &data.PriceChange24h,
				Change7dPct:  &data.PriceChange7d,
				High24hUSD:   &data.High24h,
				Low24hUSD:    &data.Low24h,
				Source:       "COINGECKO",
				Timestamp:    time.Now(),
			}
		}
	}

	return priceMap, nil
}

// savePriceRecord saves a crypto price using Hasura or SQL fallback
func (s *CryptoPricingService) savePriceRecord(ctx context.Context, price *types.CryptoPrice) error {
	// Note: Using SQL fallback primarily due to UUID type handling complexity
	// Hasura integration can be added when UUID parsing from map[string]interface{} is standardized
	query := `
		INSERT INTO crypto_prices (
			asset_symbol, price_usd, market_cap_usd, volume_24h_usd,
			change_1h_pct, change_24h_pct, change_7d_pct,
			high_24h_usd, low_24h_usd, source, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	return s.db.QueryRowContext(ctx, query,
		price.AssetSymbol, price.PriceUSD, price.MarketCapUSD, price.Volume24hUSD,
		price.Change1hPct, price.Change24hPct, price.Change7dPct,
		price.High24hUSD, price.Low24hUSD, price.Source, price.Timestamp,
	).Scan(&price.ID)
}

// getLatestPriceRecord retrieves the latest price using Hasura or SQL fallback
func (s *CryptoPricingService) getLatestPriceRecord(ctx context.Context, symbol string) (*types.CryptoPrice, error) {
	// Note: Using SQL fallback primarily due to UUID type handling complexity
	sqlQuery := `
		SELECT 
			id, asset_symbol, price_usd, market_cap_usd, volume_24h_usd,
			change_1h_pct, change_24h_pct, change_7d_pct,
			high_24h_usd, low_24h_usd, source, timestamp
		FROM crypto_latest_prices
		WHERE asset_symbol = $1
	`

	price := &types.CryptoPrice{}
	err := s.db.QueryRowContext(ctx, sqlQuery, symbol).Scan(
		&price.ID, &price.AssetSymbol, &price.PriceUSD, &price.MarketCapUSD, &price.Volume24hUSD,
		&price.Change1hPct, &price.Change24hPct, &price.Change7dPct,
		&price.High24hUSD, &price.Low24hUSD, &price.Source, &price.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	return price, nil
}

// getHistoricalPricesRecords retrieves historical prices using Hasura or SQL fallback
func (s *CryptoPricingService) getHistoricalPricesRecords(ctx context.Context, symbol string, from, to time.Time) ([]types.CryptoPrice, error) {
	// Note: Using SQL fallback primarily due to UUID type handling complexity
	sqlQuery := `
		SELECT 
			id, asset_symbol, price_usd, market_cap_usd, volume_24h_usd,
			change_24h_pct, source, timestamp
		FROM crypto_prices
		WHERE asset_symbol = $1
		  AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, symbol, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []types.CryptoPrice
	for rows.Next() {
		var p types.CryptoPrice
		err := rows.Scan(
			&p.ID, &p.AssetSymbol, &p.PriceUSD, &p.MarketCapUSD, &p.Volume24hUSD,
			&p.Change24hPct, &p.Source, &p.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}

	return prices, rows.Err()
}

// getActiveAssetSymbols retrieves distinct asset symbols from holdings using Hasura or SQL fallback
func (s *CryptoPricingService) getActiveAssetSymbols(ctx context.Context) ([]string, error) {
	// Note: Using SQL fallback primarily for simplicity
	sqlQuery := `
		SELECT DISTINCT asset_symbol
		FROM crypto_holdings
		WHERE quantity > 0
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}
		symbols = append(symbols, symbol)
	}

	return symbols, rows.Err()
}

// calculatePortfolioValueFromHoldings calculates total portfolio value using Hasura or SQL fallback
func (s *CryptoPricingService) calculatePortfolioValueFromHoldings(ctx context.Context, clientID string) (float64, error) {
	// Note: Using SQL fallback primarily due to complex JOIN logic
	sqlQuery := `
		SELECT 
			h.asset_symbol,
			h.quantity,
			p.price_usd
		FROM crypto_holdings h
		JOIN crypto_wallets w ON h.wallet_id = w.id
		LEFT JOIN crypto_latest_prices p ON h.asset_symbol = p.asset_symbol
		WHERE w.client_id = $1
		  AND h.quantity > 0
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, clientID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	totalValue := 0.0
	for rows.Next() {
		var symbol string
		var quantity float64
		var price sql.NullFloat64

		if err := rows.Scan(&symbol, &quantity, &price); err != nil {
			continue
		}

		if price.Valid {
			totalValue += quantity * price.Float64
		}
	}

	return totalValue, rows.Err()
}
