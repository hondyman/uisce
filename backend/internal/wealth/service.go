package wealth

import (

	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/platform"
	_ "github.com/lib/pq"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Client represents a wealth management client
type Client struct {
	ID             string    `json:"id"`
	ClientCode     string    `json:"client_code"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	DateOfBirth    *string   `json:"date_of_birth,omitempty"`
	Email          *string   `json:"email,omitempty"`
	RiskTolerance  string    `json:"risk_tolerance"`
	NetWorth       *float64  `json:"net_worth,omitempty"`
	AnnualIncome   *float64  `json:"annual_income,omitempty"`
	KYCStatus      string    `json:"kyc_status"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Portfolio represents an investment portfolio
type Portfolio struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"client_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	PortfolioType   string    `json:"portfolio_type"`
	BaseCurrency    string    `json:"base_currency"`
	InceptionDate   string    `json:"inception_date"`
	BenchmarkSymbol *string   `json:"benchmark_symbol,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WealthService handles wealth management operations
type WealthService struct {
	tenantManager *platform.TenantDBManager
}

// NewWealthService creates a new wealth service
func NewWealthService(tm *platform.TenantDBManager) (*WealthService, error) {
	return &WealthService{tenantManager: tm}, nil
}

// GetClients retrieves all clients
func (s *WealthService) GetClients(tenantID string) ([]Client, error) {
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, client_code, first_name, last_name, date_of_birth, email, 
		       risk_tolerance, net_worth, annual_income, kyc_status, status, 
		       created_at, updated_at
		FROM wealth.clients
		WHERE status = 'ACTIVE'
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		err := rows.Scan(
			&c.ID, &c.ClientCode, &c.FirstName, &c.LastName, &c.DateOfBirth, &c.Email,
			&c.RiskTolerance, &c.NetWorth, &c.AnnualIncome, &c.KYCStatus, &c.Status,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}

	return clients, nil
}

// CreateClient creates a new client
func (s *WealthService) CreateClient(tenantID string, c *Client) error {
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	c.ID = uuid.New().String()
	c.Status = "ACTIVE"
	c.KYCStatus = "PENDING_REVIEW"
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	query := `
		INSERT INTO wealth.clients (id, client_code, first_name, last_name, date_of_birth, email,
		                     risk_tolerance, net_worth, annual_income, kyc_status, status,
		                     created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = db.Exec(query,
		c.ID, c.ClientCode, c.FirstName, c.LastName, c.DateOfBirth, c.Email,
		c.RiskTolerance, c.NetWorth, c.AnnualIncome, c.KYCStatus, c.Status,
		c.CreatedAt, c.UpdatedAt,
	)

	return err
}

// GetPortfolios retrieves all portfolios for a client
func (s *WealthService) GetPortfolios(tenantID string, clientID string) ([]Portfolio, error) {
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, client_id, name, description, portfolio_type, base_currency,
		       inception_date, benchmark_symbol, is_active, created_at, updated_at
		FROM wealth.portfolios
		WHERE client_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []Portfolio
	for rows.Next() {
		var p Portfolio
		err := rows.Scan(
			&p.ID, &p.ClientID, &p.Name, &p.Description, &p.PortfolioType,
			&p.BaseCurrency, &p.InceptionDate, &p.BenchmarkSymbol, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}

	return portfolios, nil
}

// CreatePortfolio creates a new portfolio
func (s *WealthService) CreatePortfolio(tenantID string, p *Portfolio) error {
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	p.ID = uuid.New().String()
	p.IsActive = true
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	query := `
		INSERT INTO wealth.portfolios (id, client_id, name, description, portfolio_type,
		                        base_currency, inception_date, benchmark_symbol,
		                        is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.Exec(query,
		p.ID, p.ClientID, p.Name, p.Description, p.PortfolioType,
		p.BaseCurrency, p.InceptionDate, p.BenchmarkSymbol,
		p.IsActive, p.CreatedAt, p.UpdatedAt,
	)

	return err
}

// HTTP Handlers

// HTTP Handlers

// HandleGetClients handles GET /api/wealth/clients
func (s *WealthService) HandleGetClients(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	clients, err := s.GetClients(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

// HandleCreateClient handles POST /api/wealth/clients
func (s *WealthService) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var client Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.CreateClient(tenantID, &client); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

// HandleGetPortfolios handles GET /api/wealth/portfolios?client_id={id}
func (s *WealthService) HandleGetPortfolios(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}

	portfolios, err := s.GetPortfolios(tenantID, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolios)
}

// HandleCreatePortfolio handles POST /api/wealth/portfolios
func (s *WealthService) HandleCreatePortfolio(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var portfolio Portfolio
	if err := json.NewDecoder(r.Body).Decode(&portfolio); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.CreatePortfolio(tenantID, &portfolio); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(portfolio)
}
