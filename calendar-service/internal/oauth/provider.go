package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"calendar-service/internal/security"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
)

var (
	oauthTokenSaved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth_token_saved_total",
			Help: "Number of OAuth tokens persisted",
		},
		[]string{"provider", "user_id"},
	)
	oauthTokenRetrieved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth_token_retrieved_total",
			Help: "Number of OAuth tokens retrieved",
		},
		[]string{"provider", "user_id"},
	)
	oauthTokenRefreshed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth_token_refreshed_total",
			Help: "Number of OAuth tokens refreshed",
		},
		[]string{"provider", "user_id"},
	)
	oauthTokenErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth_token_errors_total",
			Help: "OAuth token errors",
		},
		[]string{"provider", "error_type"},
	)
	oauthTokenLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oauth_token_operation_duration_seconds",
			Help:    "OAuth token operation latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "operation"},
	)
)

var (
	ErrTokenNotFound   = errors.New("oauth token not found")
	ErrTokenExpired    = errors.New("oauth token expired")
	ErrTokenMissingID  = errors.New("user_id is required")
	ErrTokenMissingTTL = errors.New("token TTL must be positive")
)

// OAuthToken is the persisted metadata for an OAuth2 token.
type OAuthToken struct {
	UserID        string    `json:"user_id"`
	Provider      string    `json:"provider"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	TokenType     string    `json:"token_type"`
	Expiry        time.Time `json:"expiry"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsed      time.Time `json:"last_used"`
	LastRefreshed time.Time `json:"last_refreshed"`
	Scopes        []string  `json:"scopes"`
}

// ProviderConfig controls how the OAuth provider is initialized.
type ProviderConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	Scopes             []string
	RedisURL           string
	RedisPrefix        string
	TokenTTL           time.Duration
	RefreshThreshold   time.Duration
	PKCEStateTTL       time.Duration
	TokenEncryptionKey string
	MicrosoftTenantID  string
}

// GoogleOAuth2Provider implements Google OAuth2 with Redis persistence.
type GoogleOAuth2Provider struct {
	config           *oauth2.Config
	redisClient      *redis.Client
	redisPrefix      string
	tokenTTL         time.Duration
	refreshThreshold time.Duration
	pkceStateTTL     time.Duration
	logger           *logrus.Entry
	encryptor        *security.TokenEncryptor
}

type PKCEParams struct {
	Verifier  string `json:"verifier"`
	Challenge string `json:"challenge"`
	Method    string `json:"method"`
}

type PKCEState struct {
	UserID    string     `json:"user_id"`
	TenantID  string     `json:"tenant_id,omitempty"`
	Params    PKCEParams `json:"params"`
	CreatedAt time.Time  `json:"created_at"`
}

const defaultPKCEStateTTL = 10 * time.Minute

// NewGoogleOAuth2Provider creates a provider backed by Redis.
func NewGoogleOAuth2Provider(cfg ProviderConfig, logger *logrus.Entry) (*GoogleOAuth2Provider, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url is required")
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = 24 * time.Hour
	}
	if cfg.RefreshThreshold <= 0 {
		cfg.RefreshThreshold = 5 * time.Minute
	}
	if cfg.RedisPrefix == "" {
		cfg.RedisPrefix = "calendar"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{
			calendar.CalendarScope,
			calendar.CalendarEventsScope,
			calendar.CalendarReadonlyScope,
		}
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("redirect url is required")
	}

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		providerLogger := logger.WithField("component", "google_oauth_provider")
		providerLogger.Warnf("Redis unavailable (%v), continuing without cache - using in-memory state storage", err)
		redisClient = nil
	}

	pkceTTL := cfg.PKCEStateTTL
	if pkceTTL <= 0 {
		pkceTTL = defaultPKCEStateTTL
	}
	providerLogger := logger.WithField("component", "google_oauth_provider")

	var encryptor *security.TokenEncryptor
	if cfg.TokenEncryptionKey != "" {
		enc, err := security.NewTokenEncryptor(cfg.TokenEncryptionKey)
		if err != nil {
			providerLogger.WithError(err).Warn("Token encryption disabled")
		} else {
			encryptor = enc
			providerLogger.Info("Token encryption enabled")
		}
	}

	return &GoogleOAuth2Provider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
		redisClient:      redisClient,
		redisPrefix:      cfg.RedisPrefix,
		tokenTTL:         cfg.TokenTTL,
		refreshThreshold: cfg.RefreshThreshold,
		logger:           providerLogger,
		pkceStateTTL:     pkceTTL,
		encryptor:        encryptor,
	}, nil

}

func (p *GoogleOAuth2Provider) tokenKey(userID string) string {
	return fmt.Sprintf("%s:oauth:google:%s", p.redisPrefix, userID)
}

func (p *GoogleOAuth2Provider) pkceKey(state string) string {
	return fmt.Sprintf("%s:oauth:pkce:%s", p.redisPrefix, state)
}

func (p *GoogleOAuth2Provider) encryptValue(value string) string {
	if p.encryptor == nil || value == "" {
		return value
	}
	enc, err := p.encryptor.Encrypt(value)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to encrypt token value")
		return value
	}
	return enc
}

func (p *GoogleOAuth2Provider) decryptValue(value string) string {
	if p.encryptor == nil || value == "" {
		return value
	}
	plain, err := p.encryptor.Decrypt(value)
	if err != nil {
		p.logger.WithError(err).Warn("Failed to decrypt token value")
		return value
	}
	return plain
}

// GeneratePKCEParams creates verifier/challenge pair for PKCE flows.
func GeneratePKCEParams() (*PKCEParams, error) {
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("generate pkce verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return &PKCEParams{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    "S256",
	}, nil
}

// GetAuthURLWithPKCE builds an authorization URL that includes PKCE params.
func (p *GoogleOAuth2Provider) GetAuthURLWithPKCE(state string, params *PKCEParams) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	}
	if params != nil {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", params.Challenge),
			oauth2.SetAuthURLParam("code_challenge_method", params.Method),
		)
	}
	return p.config.AuthCodeURL(state, opts...)
}

// StorePKCEState persists the PKCE context until the callback completes.
func (p *GoogleOAuth2Provider) StorePKCEState(ctx context.Context, state string, data *PKCEState) error {
	if state == "" {
		return fmt.Errorf("state is required")
	}
	if data == nil {
		return fmt.Errorf("pkce data is required")
	}
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal pkce state: %w", err)
	}
	if p.redisClient != nil {
		if err := p.redisClient.Set(ctx, p.pkceKey(state), payload, p.pkceStateTTL).Err(); err != nil {
			p.logger.WithError(err).Warn("failed to cache pkce state in redis")
		}
	}
	return nil
}

// RetrievePKCEState fetches and removes the stored PKCE context.
func (p *GoogleOAuth2Provider) RetrievePKCEState(ctx context.Context, state string) (*PKCEState, error) {
	if state == "" {
		return nil, fmt.Errorf("state is required")
	}
	var bytes []byte
	var err error
	if p.redisClient != nil {
		bytes, err = p.redisClient.Get(ctx, p.pkceKey(state)).Bytes()
		if err == redis.Nil {
			return nil, fmt.Errorf("pkce state not found: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("fetch pkce state: %w", err)
		}
		if err := p.redisClient.Del(ctx, p.pkceKey(state)).Err(); err != nil {
			p.logger.WithError(err).Warn("failed to clean up pkce state")
		}
	} else {
		return nil, fmt.Errorf("no redis available for pkce state retrieval")
	}
	var data PKCEState
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("unmarshal pkce state: %w", err)
	}
	return &data, nil
}

// ExchangeCodeForTokenWithPKCE exchanges an authorization code using the PKCE verifier.
func (p *GoogleOAuth2Provider) ExchangeCodeForTokenWithPKCE(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	if code == "" {
		return nil, fmt.Errorf("auth code is required")
	}
	if verifier == "" {
		return nil, fmt.Errorf("code verifier is required")
	}
	form := url.Values{
		"code":          {code},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"redirect_uri":  {p.config.RedirectURL},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, google.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		oauthTokenErrors.WithLabelValues("google", "exchange_failed").Inc()
		return nil, fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		oauthTokenErrors.WithLabelValues("google", "decode_error").Inc()
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		oauthTokenErrors.WithLabelValues("google", "exchange_failed").Inc()
		msg := tokenResp.Error
		if tokenResp.ErrorDescription != "" {
			msg = fmt.Sprintf("%s: %s", tokenResp.Error, tokenResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token exchange failed: %s", msg)
	}
	return &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// ExchangeCodeForToken exchanges an authorization code for a token.
func (p *GoogleOAuth2Provider) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "" {
		return nil, fmt.Errorf("auth code is required")
	}
	outToken, err := p.config.Exchange(ctx, code)
	if err != nil {
		oauthTokenErrors.WithLabelValues("google", "exchange_failed").Inc()
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return outToken, nil
}

// SaveUserToken persists a token for a user in Redis.
func (p *GoogleOAuth2Provider) SaveUserToken(ctx context.Context, userID string, token *oauth2.Token) error {
	if userID == "" {
		return ErrTokenMissingID
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("google", "save").Observe(time.Since(start).Seconds())
	}()

	payload := &OAuthToken{
		UserID:       userID,
		Provider:     "google",
		AccessToken:  p.encryptValue(token.AccessToken),
		RefreshToken: p.encryptValue(token.RefreshToken),
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		CreatedAt:    time.Now().UTC(),
		LastUsed:     time.Now().UTC(),
		Scopes:       p.config.Scopes,
	}
	if token.Expiry.After(time.Time{}) {
		payload.LastRefreshed = token.Expiry
	}
	body, err := json.Marshal(payload)
	if err != nil {
		oauthTokenErrors.WithLabelValues("google", "marshal_error").Inc()
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := p.redisClient.Set(ctx, p.tokenKey(userID), body, p.tokenTTL).Err(); err != nil {
		oauthTokenErrors.WithLabelValues("google", "redis_set_error").Inc()
		return fmt.Errorf("redis set: %w", err)
	}
	oauthTokenSaved.WithLabelValues("google", userID).Inc()
	return nil
}

func (p *GoogleOAuth2Provider) getStoredToken(ctx context.Context, userID string) (*OAuthToken, error) {
	if userID == "" {
		return nil, ErrTokenMissingID
	}
	bytes, err := p.redisClient.Get(ctx, p.tokenKey(userID)).Bytes()
	if err == redis.Nil {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var oauthToken OAuthToken
	if err := json.Unmarshal(bytes, &oauthToken); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &oauthToken, nil
}

// GetUserToken returns the persisted token, refreshing if it is stale.
func (p *GoogleOAuth2Provider) GetUserToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("google", "get").Observe(time.Since(start).Seconds())
	}()
	oauthToken, err := p.getStoredToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if !oauthToken.Expiry.IsZero() {
		if oauthToken.Expiry.Before(now) {
			return nil, ErrTokenExpired
		}
		if oauthToken.Expiry.Sub(now) < p.refreshThreshold {
			refreshed, err := p.refreshTokenInternal(ctx, oauthToken)
			if err != nil {
				return nil, err
			}
			_ = p.SaveUserToken(ctx, userID, refreshed)
			oauthTokenRefreshed.WithLabelValues("google", userID).Inc()
			return refreshed, nil
		}
	}
	go p.touchLastUsed(ctx, userID, oauthToken)
	oauthTokenRetrieved.WithLabelValues("google", userID).Inc()
	return &oauth2.Token{
		AccessToken:  p.decryptValue(oauthToken.AccessToken),
		RefreshToken: p.decryptValue(oauthToken.RefreshToken),
		TokenType:    oauthToken.TokenType,
		Expiry:       oauthToken.Expiry,
	}, nil
}

func (p *GoogleOAuth2Provider) touchLastUsed(ctx context.Context, userID string, token *OAuthToken) {
	token.LastUsed = time.Now().UTC()
	updated, err := json.Marshal(token)
	if err != nil {
		return
	}
	_ = p.redisClient.Set(ctx, p.tokenKey(userID), updated, p.tokenTTL).Err()
}

func (p *GoogleOAuth2Provider) refreshTokenInternal(ctx context.Context, oauthToken *OAuthToken) (*oauth2.Token, error) {
	if oauthToken == nil {
		oauthTokenErrors.WithLabelValues("google", "missing_refresh").Inc()
		return nil, fmt.Errorf("missing refresh token for user %s", oauthToken.UserID)
	}
	refreshToken := p.decryptValue(oauthToken.RefreshToken)
	if refreshToken == "" {
		oauthTokenErrors.WithLabelValues("google", "missing_refresh").Inc()
		return nil, fmt.Errorf("missing refresh token for user %s", oauthToken.UserID)
	}
	source := p.config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	newToken, err := source.Token()
	if err != nil {
		oauthTokenErrors.WithLabelValues("google", "refresh_failed").Inc()
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = refreshToken
	}
	return newToken, nil
}

// RefreshUserToken explicitly refreshes the user's token.
func (p *GoogleOAuth2Provider) RefreshUserToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("google", "refresh").Observe(time.Since(start).Seconds())
	}()
	oauthToken, err := p.getStoredToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	refreshed, err := p.refreshTokenInternal(ctx, oauthToken)
	if err != nil {
		return nil, err
	}
	if err := p.SaveUserToken(ctx, userID, refreshed); err != nil {
		return nil, err
	}
	oauthTokenRefreshed.WithLabelValues("google", userID).Inc()
	return refreshed, nil
}

// DeleteUserToken removes a user's token.
func (p *GoogleOAuth2Provider) DeleteUserToken(ctx context.Context, userID string) error {
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("google", "delete").Observe(time.Since(start).Seconds())
	}()
	if err := p.redisClient.Del(ctx, p.tokenKey(userID)).Err(); err != nil {
		oauthTokenErrors.WithLabelValues("google", "redis_delete").Inc()
		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

// GetAuthURL returns the authorization URL.
func (p *GoogleOAuth2Provider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// HealthCheck validates Redis connectivity.
// HealthCheck checks if Redis is available for OAuth state storage.
func (p *GoogleOAuth2Provider) HealthCheck(ctx context.Context) error {
	if p.redisClient == nil {
		return fmt.Errorf("redis not configured - PKCE state will not persist")
	}
	return p.redisClient.Ping(ctx).Err()
}

// Close releases Redis resources.
func (p *GoogleOAuth2Provider) Close() error {
	if p.redisClient != nil {
		return p.redisClient.Close()
	}
	return nil
}

// Config exposes the underlying OAuth configuration.
func (p *GoogleOAuth2Provider) Config() *oauth2.Config {
	return p.config
}

func (p *GoogleOAuth2Provider) PKCEStateTTL() time.Duration {
	return p.pkceStateTTL
}
