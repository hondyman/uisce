package fix

import (
	"fmt"
	"log"
	"os"

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
		settings.GlobalSettings().Set(config.SocketAcceptHost, "0.0.0.0")
		settings.GlobalSettings().Set(config.SocketAcceptPort, "8980")
		settings.GlobalSettings().Set(config.BeginString, "FIX.4.4")
		settings.GlobalSettings().Set(config.DefaultApplVerID, "FIX.4.4")
	}

	acceptor, err := adapter.CreateAcceptor(settings)
	if err != nil {
		return nil, fmt.Errorf("create acceptor: %w", err)
	}

	return &Server{acceptor: acceptor, adapter: adapter}, nil
}

func (s *Server) Start() error {
	if err := s.acceptor.Start(); err != nil {
		return fmt.Errorf("start acceptor: %w", err)
	}
	log.Println("[FIX] Acceptor started")
	return nil
}

func (s *Server) Stop() {
	s.acceptor.Stop()
	log.Println("[FIX] Acceptor stopped")
}

func ConfigExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
