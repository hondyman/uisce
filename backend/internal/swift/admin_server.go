package swift

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AdminServer is a localhost-only HTTP server for administration.
type AdminServer struct {
	addr    string
	token   string
	adapter *Adapter
}

// NewAdminServer creates a new admin server.
func NewAdminServer(addr, token string) *AdminServer {
	if addr == "" {
		addr = "127.0.0.1:8982"
	}
	return &AdminServer{
		addr:  addr,
		token: token,
	}
}

// Start runs the server on the configured address.
func (s *AdminServer) Start(ctx context.Context) error {
	if !strings.HasPrefix(s.addr, "127.0.0.1:") {
		return fmt.Errorf("admin server must bind to 127.0.0.1 for security")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/channels", s.authWrap(s.handleChannels))
	mux.HandleFunc("/channels/", s.authWrap(s.handleChannelAction))

	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

func (s *AdminServer) authWrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Swift-Admin-Token") != s.token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *AdminServer) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode([]string{})
}

func (s *AdminServer) handleChannelAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/channels/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	action := parts[1]
	switch action {
	case "health":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case "send":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			MsgType string `json:"msg_type"`
			Raw     string `json:"raw"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "cancel":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TransactionRef string `json:"transaction_ref"`
			Reason         string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}
