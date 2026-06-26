package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PricingProvider is the interface for market data providers
type PricingProvider interface {
	GetPrice(ctx context.Context, ticker string) (float64, error)
	GetFXRate(ctx context.Context, pair string) (float64, error) // e.g., "EURUSD"
	Name() string
}

// PriceQuote represents a price with metadata
type PriceQuote struct {
	Ticker    string    `json:"ticker"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// --- Yahoo Finance Provider (Free, delayed) ---

type YahooFinanceProvider struct {
	client *http.Client
}

func NewYahooFinanceProvider() *YahooFinanceProvider {
	return &YahooFinanceProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *YahooFinanceProvider) Name() string {
	return "Yahoo Finance"
}

func (p *YahooFinanceProvider) GetPrice(ctx context.Context, ticker string) (float64, error) {
	// Yahoo Finance API is not officially supported, but we can use chart API
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", ticker)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("yahoo finance returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if len(result.Chart.Result) == 0 {
		return 0, fmt.Errorf("no price data for ticker %s", ticker)
	}
	
	return result.Chart.Result[0].Meta.RegularMarketPrice, nil
}

func (p *YahooFinanceProvider) GetFXRate(ctx context.Context, pair string) (float64, error) {
	// Yahoo uses ticker format: EURUSD=X for fx pairs
	ticker := pair + "=X"
	return p.GetPrice(ctx, ticker)
}

// --- Alpha Vantage Provider (Free tier with API key) ---

type AlphaVantageProvider struct {
	apiKey string
	client *http.Client
}

func NewAlphaVantageProvider(apiKey string) *AlphaVantageProvider {
	return &AlphaVantageProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *AlphaVantageProvider) Name() string {
	return "Alpha Vantage"
}

func (p *AlphaVantageProvider) GetPrice(ctx context.Context, ticker string) (float64, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		ticker, p.apiKey)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}
	
	var result struct {
		GlobalQuote struct {
			Price string `json:"05. price"`
		} `json:"Global Quote"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if result.GlobalQuote.Price == "" {
		return 0, fmt.Errorf("no price data for ticker %s", ticker)
	}
	
	var price float64
	if _, err := fmt.Sscanf(result.GlobalQuote.Price, "%f", &price); err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}
	
	return price, nil
}

func (p *AlphaVantageProvider) GetFXRate(ctx context.Context, pair string) (float64, error) {
	// Alpha Vantage FX endpoint
	if len(pair) != 6 {
		return 0, fmt.Errorf("invalid FX pair format, expected 6 chars like EURUSD")
	}
	
	from := pair[:3]
	to := pair[3:]
	
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=CURRENCY_EXCHANGE_RATE&from_currency=%s&to_currency=%s&apikey=%s",
		from, to, p.apiKey)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch fx rate: %w", err)
	}
	defer resp.Body.Close()
	
	var result struct {
		RealtimeCurrencyExchangeRate struct {
			ExchangeRate string `json:"5. Exchange Rate"`
		} `json:"Realtime Currency Exchange Rate"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}
	
	var rate float64
	if _, err := fmt.Sscanf(result.RealtimeCurrencyExchangeRate.ExchangeRate, "%f", &rate); err != nil {
		return 0, fmt.Errorf("failed to parse exchange rate: %w", err)
	}
	
	return rate, nil
}

// --- Bloomberg Provider (Placeholder for future) ---

type BloombergProvider struct {
	// TODO: Add Bloomberg API client
}

func NewBloombergProvider() *BloombergProvider {
	return &BloombergProvider{}
}

func (p *BloombergProvider) Name() string {
	return "Bloomberg"
}

func (p *BloombergProvider) GetPrice(ctx context.Context, ticker string) (float64, error) {
	// TODO: Implement Bloomberg BLPAPI integration
	return 0, fmt.Errorf("Bloomberg provider not yet implemented - swap in Bloomberg BLPAPI here")
}

func (p *BloombergProvider) GetFXRate(ctx context.Context, pair string) (float64, error) {
	// TODO: Implement Bloomberg FX rate lookup
	return 0, fmt.Errorf("Bloomberg provider not yet implemented")
}
