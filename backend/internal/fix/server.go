package fix

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
)

type Server struct {
	acceptor *quickfix.Acceptor
	adapter  *Adapter
}

func NewServer(adapter *Adapter, configPath string) (*Server, error) {
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

	var acceptor *quickfix.Acceptor
	if adapter == nil {
		acceptor = nil
	} else {
		var acc *quickfix.Acceptor
		acc, err = adapter.CreateAcceptor(settings)
		acceptor = acc
		if err != nil {
			return nil, fmt.Errorf("create acceptor: %w", err)
		}
	}

	return &Server{acceptor: acceptor, adapter: adapter}, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.acceptor != nil {
		if err := s.acceptor.Start(); err != nil {
			return fmt.Errorf("start acceptor: %w", err)
		}
		log.Println("[FIX] Acceptor started")
	} else {
		log.Println("[FIX] Acceptor not configured (nil adapter)")
	}

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

func (s *Server) Stop() {
	if s.acceptor != nil {
		s.acceptor.Stop()
		log.Println("[FIX] Acceptor stopped")
	}
}

func ConfigExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
