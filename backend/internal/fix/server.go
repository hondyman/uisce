package fix

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
)

// Server hosts the quickfix.Acceptor (the Layer 1 FIX driver from
// HANDOFF_FIX_OVER_PIPELINE.md §6). The acceptor process is the sole
// owner of FIX TCP sockets in this architecture — Temporal workflows
// never instantiate quickfix sessions; they talk to the acceptor via
// the internal admin API on 127.0.0.1:8981.
type Server struct {
	acceptor    *quickfix.Acceptor
	adapter     *Adapter
	adminServer *AdminServer
}

// NewServer creates the acceptor. configPath, when non-empty, is parsed
// as a quickfix configuration file; otherwise an in-process settings
// object is built from FIX_ACCEPTOR_PORT / defaults.
//
// adapter is the FIX-over-pipeline adapter (NewPipelineAdapter) or nil
// for legacy in-process compliance evaluation.
//
// db is the *sql.DB used by the Postgres message store (Amendment 3)
// and the append-only fix_session_log audit. Callers using *sqlx.DB
// should pass `sqlxDB.DB` to unwrap. When nil, the in-memory store is
// used and a warning is logged at startup — see §8 of the handoff for
// when that's acceptable.
//
// adminAddr is the address (host:port) for the internal admin listener,
// e.g. "127.0.0.1:8981". Empty disables the admin API (legacy mode).
//
// adminToken is the shared-secret expected in the X-Fix-Admin-Token
// header. Required when adminAddr is non-empty; ignored otherwise.
func NewServer(adapter *Adapter, configPath string, db *sql.DB, adminAddr string, adminToken string) (*Server, error) {
	var settings *quickfix.Settings
	var err error

	if configPath != "" && ConfigExists(configPath) {
		file, openErr := os.Open(configPath)
		if openErr != nil {
			return nil, fmt.Errorf("open config: %w", openErr)
		}
		settings, err = quickfix.ParseSettings(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("parse settings: %w", err)
		}
	} else {
		settings = quickfix.NewSettings()
		settings.GlobalSettings().Set(config.BeginString, "FIX.4.4")
		settings.GlobalSettings().Set(config.DefaultApplVerID, "FIX.4.4")

		port := "8980"
		if envPort := os.Getenv("FIX_ACCEPTOR_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
				port = envPort
			}
		}
		settings.GlobalSettings().Set(config.SocketAcceptHost, "0.0.0.0")
		settings.GlobalSettings().Set(config.SocketAcceptPort, port)
	}

	if os.Getenv("FIX_DEMO_AGENT") == "true" {
		acceptPort := os.Getenv("FIX_ACCEPTOR_PORT")
		if acceptPort == "" {
			acceptPort = "8980"
		}
		if err := ApplyDemoAcceptorSession(settings, acceptPort); err != nil {
			return nil, fmt.Errorf("demo session: %w", err)
		}
	}

	var acceptor *quickfix.Acceptor
	if adapter == nil {
		acceptor = nil
	} else {
		acc, createErr := adapter.CreateAcceptor(settings, db)
		acceptor = acc
		if createErr != nil {
			return nil, fmt.Errorf("create acceptor: %w", createErr)
		}
	}

	srv := &Server{acceptor: acceptor, adapter: adapter}

	if adminAddr != "" {
		if adminToken == "" {
			return nil, fmt.Errorf("adminAddr is set but adminToken is empty; refusing to start an unauthenticated admin listener")
		}
		admin, adminErr := NewAdminServer(adminAddr, adminToken, adapter)
		if adminErr != nil {
			return nil, fmt.Errorf("create admin server: %w", adminErr)
		}
		srv.adminServer = admin
	}

	return srv, nil
}

// Start launches the acceptor and (if configured) the admin API.
func (s *Server) Start(ctx context.Context) error {
	if s.acceptor != nil {
		if err := s.acceptor.Start(); err != nil {
			return fmt.Errorf("start acceptor: %w", err)
		}
		log.Println("[FIX] Acceptor started")
	} else {
		log.Println("[FIX] Acceptor not configured (nil adapter)")
	}

	if s.adminServer != nil {
		if err := s.adminServer.Start(ctx); err != nil {
			return fmt.Errorf("start admin server: %w", err)
		}
		log.Printf("[FIX] Admin API listening on %s", s.adminServer.Addr())
	}

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop shuts down the acceptor and the admin API.
func (s *Server) Stop() {
	if s.acceptor != nil {
		s.acceptor.Stop()
		log.Println("[FIX] Acceptor stopped")
	}
	if s.adminServer != nil {
		s.adminServer.Stop()
		log.Println("[FIX] Admin API stopped")
	}
}

func ConfigExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
