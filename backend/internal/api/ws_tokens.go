package api

import (
	"encoding/json"
	"fmt"
	jwt "github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

// getWsToken issues a short-lived signed token for websocket usage.
func (s *Server) getWsToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID   string `json:"jobId"`
		Purpose string `json:"purpose"`
		TTL     int    `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.JobID == "" || req.Purpose == "" {
		http.Error(w, "jobId and purpose are required", http.StatusBadRequest)
		return
	}
	if req.TTL <= 0 || req.TTL > 3600 {
		req.TTL = 60
	}

	claims := jwt.MapClaims{
		"job_id":  req.JobID,
		"purpose": req.Purpose,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Duration(req.TTL) * time.Second).Unix(),
	}

	var signed string
	var err error
	if s.SecMgr != nil {
		signed, err = s.SecMgr.SignToken(claims)
	} else {
		secret := []byte(getEnv("JWT_SECRET", "dev-jwt-secret-key-change-in-production"))
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err = token.SignedString(secret)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign token: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": signed})
}

// validateWsToken validates a JWT token for websocket usage and ensures it is scoped to the provided jobId.
func (s *Server) validateWsToken(tokenString string, jobId string) (map[string]interface{}, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}
	if len(tokenString) >= 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	var claims jwt.MapClaims
	var err error
	if s.SecMgr != nil {
		var parsed map[string]interface{}
		parsed, err = s.SecMgr.ParseToken(tokenString)
		if err == nil {
			// convert parsed to jwt.MapClaims-like map
			claims = jwt.MapClaims{}
			for k, v := range parsed {
				claims[k] = v
			}
		}
	} else {
		secret := []byte(getEnv("JWT_SECRET", "dev-jwt-secret-key-change-in-production"))
		token, err2 := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})
		if err2 != nil {
			return nil, err2
		}
		if !token.Valid {
			return nil, fmt.Errorf("invalid token")
		}
		var ok bool
		claims, ok = token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}
	}

	// Verify job_id and purpose
	if j, ok := claims["job_id"].(string); !ok || j != jobId {
		return nil, fmt.Errorf("token not valid for job_id")
	}
	if p, ok := claims["purpose"].(string); !ok || p != "profiler" {
		return nil, fmt.Errorf("invalid token purpose")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return nil, fmt.Errorf("token expired")
		}
	}

	out := make(map[string]interface{})
	for k, v := range claims {
		out[k] = v
	}
	return out, nil
}
