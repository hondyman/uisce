package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	tokenSaveErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oauth_token_save_errors_total",
		Help: "Total number of token save errors",
	})
	tokenRefreshErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oauth_token_refresh_errors_total",
		Help: "Total number of token refresh errors",
	})
)

type TokenEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertextB64 string) (string, error)
}

type GoogleOAuth2Provider struct {
	config    *oauth2.Config
	redis     *redis.Client
	encryptor TokenEncryptor
}

func NewGoogleOAuth2Provider(
	clientID, clientSecret, redirectURL string,
	redisClient *redis.Client,
	encryptor TokenEncryptor,
) *GoogleOAuth2Provider {
	return &GoogleOAuth2Provider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/calendar",
				"https://www.googleapis.com/auth/calendar.events",
				"email",
				"profile",
			},
			Endpoint: google.Endpoint,
		},
		redis:     redisClient,
		encryptor: encryptor,
	}
}

func (p *GoogleOAuth2Provider) Config() *oauth2.Config {
	return p.config
}

func GeneratePKCEParams() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)

	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])

	return verifier, challenge, nil
}

func (p *GoogleOAuth2Provider) GetAuthURLWithPKCE(state, challenge string) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	}

	if challenge != "" {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}

	return p.config.AuthCodeURL(state, opts...)
}

func (p *GoogleOAuth2Provider) Exchange(ctx context.Context, code string, verifier string) (*oauth2.Token, error) {
	opts := []oauth2.AuthCodeOption{}
	if verifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", verifier))
	}
	return p.config.Exchange(ctx, code, opts...)
}

func (p *GoogleOAuth2Provider) SaveUserToken(ctx context.Context, userID string, token *oauth2.Token) error {
	storedToken := *token

	if p.encryptor != nil {
		var err error
		if storedToken.AccessToken != "" {
			storedToken.AccessToken, err = p.encryptor.Encrypt(storedToken.AccessToken)
			if err != nil {
				tokenSaveErrors.Inc()
				return fmt.Errorf("encrypt access token: %w", err)
			}
		}
		if storedToken.RefreshToken != "" {
			storedToken.RefreshToken, err = p.encryptor.Encrypt(storedToken.RefreshToken)
			if err != nil {
				tokenSaveErrors.Inc()
				return fmt.Errorf("encrypt refresh token: %w", err)
			}
		}
	}

	data, err := json.Marshal(storedToken)
	if err != nil {
		tokenSaveErrors.Inc()
		return fmt.Errorf("marshal token: %w", err)
	}

	key := fmt.Sprintf("oauth:google:%s", userID)
	if err := p.redis.Set(ctx, key, data, 0).Err(); err != nil {
		tokenSaveErrors.Inc()
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (p *GoogleOAuth2Provider) GetUserToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	key := fmt.Sprintf("oauth:google:%s", userID)
	data, err := p.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	if p.encryptor != nil {
		if token.AccessToken != "" {
			token.AccessToken, err = p.encryptor.Decrypt(token.AccessToken)
			if err != nil {
				return nil, fmt.Errorf("decrypt access token: %w", err)
			}
		}
		if token.RefreshToken != "" {
			token.RefreshToken, err = p.encryptor.Decrypt(token.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("decrypt refresh token: %w", err)
			}
		}
	}

	if !token.Valid() && token.RefreshToken != "" {
		newToken, err := p.config.TokenSource(ctx, &token).Token()
		if err != nil {
			tokenRefreshErrors.Inc()
			return nil, fmt.Errorf("refresh token: %w", err)
		}
		if err := p.SaveUserToken(ctx, userID, newToken); err != nil {
			fmt.Printf("failed to save refreshed token: %v\n", err)
		}
		return newToken, nil
	}

	return &token, nil
}
