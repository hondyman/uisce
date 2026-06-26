package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"calendar-service/internal/security"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// MicrosoftOAuth2Provider implements Microsoft OAuth2 with Redis persistence.
type MicrosoftOAuth2Provider struct {
	config           *oauth2.Config
	redisClient      *redis.Client
	redisPrefix      string
	tokenTTL         time.Duration
	refreshThreshold time.Duration
	pkceStateTTL     time.Duration
	logger           *logrus.Entry
	encryptor        *security.TokenEncryptor
}

// NewMicrosoftOAuth2Provider creates a provider backed by Redis.
func NewMicrosoftOAuth2Provider(cfg ProviderConfig, logger *logrus.Entry) (*MicrosoftOAuth2Provider, error) {
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
			"Calendars.ReadWrite",
			"offline_access",
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
		providerLogger := logger.WithField("component", "microsoft_oauth_provider")
		providerLogger.Warnf("Redis unavailable (%v), continuing without cache - using in-memory state storage", err)
		redisClient = nil
	}

	pkceTTL := cfg.PKCEStateTTL
	if pkceTTL <= 0 {
		pkceTTL = 10 * time.Minute
	}
	providerLogger := logger.WithField("component", "microsoft_oauth_provider")

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

	tenantID := cfg.MicrosoftTenantID
	if tenantID == "" {
		tenantID = "common"
	}

	return &MicrosoftOAuth2Provider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			// Use configured tenant ID or "common" for multi-tenant applications
			Endpoint: microsoft.AzureADEndpoint(tenantID),
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

func (p *MicrosoftOAuth2Provider) tokenKey(userID string) string {
	return fmt.Sprintf("%s:oauth:microsoft:%s", p.redisPrefix, userID)
}

func (p *MicrosoftOAuth2Provider) pkceKey(state string) string {
	return fmt.Sprintf("%s:oauth:pkce:ms:%s", p.redisPrefix, state)
}

func (p *MicrosoftOAuth2Provider) encryptValue(value string) string {
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

func (p *MicrosoftOAuth2Provider) decryptValue(value string) string {
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

// GetAuthURLWithPKCE builds an authorization URL that includes PKCE params.
func (p *MicrosoftOAuth2Provider) GetAuthURLWithPKCE(state string, params *PKCEParams) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("prompt", "consent"), // Force consent to get refresh token
	}
	if params != nil {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", params.Challenge),
			oauth2.SetAuthURLParam("code_challenge_method", params.Method),
		)
	}
	return p.config.AuthCodeURL(state, opts...)
}

// PKCEStateTTL returns the configured TTL for PKCE state.
func (p *MicrosoftOAuth2Provider) PKCEStateTTL() time.Duration {
	return p.pkceStateTTL
}

// StorePKCEState persists the PKCE context until the callback completes.
func (p *MicrosoftOAuth2Provider) StorePKCEState(ctx context.Context, state string, data *PKCEState) error {
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
func (p *MicrosoftOAuth2Provider) RetrievePKCEState(ctx context.Context, state string) (*PKCEState, error) {
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
func (p *MicrosoftOAuth2Provider) ExchangeCodeForTokenWithPKCE(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		oauthTokenErrors.WithLabelValues("microsoft", "exchange_failed").Inc()
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
		oauthTokenErrors.WithLabelValues("microsoft", "decode_error").Inc()
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		oauthTokenErrors.WithLabelValues("microsoft", "exchange_failed").Inc()
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

// SaveUserToken persists a token for a user in Redis.
func (p *MicrosoftOAuth2Provider) SaveUserToken(ctx context.Context, userID string, token *oauth2.Token) error {
	if userID == "" {
		return ErrTokenMissingID
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("microsoft", "save").Observe(time.Since(start).Seconds())
	}()

	payload := &OAuthToken{
		UserID:       userID,
		Provider:     "microsoft",
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
		oauthTokenErrors.WithLabelValues("microsoft", "marshal_error").Inc()
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := p.redisClient.Set(ctx, p.tokenKey(userID), body, p.tokenTTL).Err(); err != nil {
		oauthTokenErrors.WithLabelValues("microsoft", "redis_set_error").Inc()
		return fmt.Errorf("redis set: %w", err)
	}
	oauthTokenSaved.WithLabelValues("microsoft", userID).Inc()
	return nil
}

func (p *MicrosoftOAuth2Provider) getStoredToken(ctx context.Context, userID string) (*OAuthToken, error) {
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
func (p *MicrosoftOAuth2Provider) GetUserToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("microsoft", "get").Observe(time.Since(start).Seconds())
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
			oauthTokenRefreshed.WithLabelValues("microsoft", userID).Inc()
			return refreshed, nil
		}
	}
	go p.touchLastUsed(ctx, userID, oauthToken)
	oauthTokenRetrieved.WithLabelValues("microsoft", userID).Inc()
	return &oauth2.Token{
		AccessToken:  p.decryptValue(oauthToken.AccessToken),
		RefreshToken: p.decryptValue(oauthToken.RefreshToken),
		TokenType:    oauthToken.TokenType,
		Expiry:       oauthToken.Expiry,
	}, nil
}

func (p *MicrosoftOAuth2Provider) touchLastUsed(ctx context.Context, userID string, token *OAuthToken) {
	token.LastUsed = time.Now().UTC()
	updated, err := json.Marshal(token)
	if err != nil {
		return
	}
	_ = p.redisClient.Set(ctx, p.tokenKey(userID), updated, p.tokenTTL).Err()
}

func (p *MicrosoftOAuth2Provider) refreshTokenInternal(ctx context.Context, oauthToken *OAuthToken) (*oauth2.Token, error) {
	if oauthToken == nil {
		oauthTokenErrors.WithLabelValues("microsoft", "missing_refresh").Inc()
		return nil, fmt.Errorf("missing refresh token for user %s", oauthToken.UserID)
	}
	refreshToken := p.decryptValue(oauthToken.RefreshToken)
	if refreshToken == "" {
		oauthTokenErrors.WithLabelValues("microsoft", "missing_refresh").Inc()
		return nil, fmt.Errorf("missing refresh token for user %s", oauthToken.UserID)
	}
	source := p.config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	newToken, err := source.Token()
	if err != nil {
		oauthTokenErrors.WithLabelValues("microsoft", "refresh_failed").Inc()
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = refreshToken
	}
	return newToken, nil
}

// DeleteUserToken removes a user's token.
func (p *MicrosoftOAuth2Provider) DeleteUserToken(ctx context.Context, userID string) error {
	start := time.Now()
	defer func() {
		oauthTokenLatency.WithLabelValues("microsoft", "delete").Observe(time.Since(start).Seconds())
	}()
	if err := p.redisClient.Del(ctx, p.tokenKey(userID)).Err(); err != nil {
		oauthTokenErrors.WithLabelValues("microsoft", "redis_delete").Inc()
		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

// Close releases Redis resources.
func (p *MicrosoftOAuth2Provider) Close() error {
	if p.redisClient != nil {
		return p.redisClient.Close()
	}
	return nil
}
