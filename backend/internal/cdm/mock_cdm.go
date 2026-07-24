package cdm

// MockPayout mimics a CDM Payout object
type MockPayout struct {
	PayerReceiver PayerReceiver `json:"payerReceiver"`
	PriceQuantity PriceQuantity `json:"priceQuantity"`
}

// MockSwap mimics a CDM InterestRateSwap
type MockSwap struct {
	PrimaryAssetClass string       `json:"primaryAssetClass"`
	Payout            MockPayout   `json:"payout"`
	EffectiveDate     Serializable `json:"effectiveDate"` // Use interface/struct to test object mapping
	TerminationDate   string       `json:"terminationDate"`
	Notional          float64      `json:"notional"`
	IsFixed           bool         `json:"isFixed"`
}

type PayerReceiver struct {
	Payer    string `json:"payer"`
	Receiver string `json:"receiver"`
}

type PriceQuantity struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type Serializable interface {
	Serialize() string
}
