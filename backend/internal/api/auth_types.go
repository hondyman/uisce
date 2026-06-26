package api

import (
	"github.com/hondyman/semlayer/backend/internal/models"
)

// Auth-related DTOs used by authentication handlers.
// Kept small and package-local to avoid cross-package coupling in tests.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RegisterRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name,omitempty"`
	Role         string `json:"role,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type AuthResponse struct {
	User         models.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
}
