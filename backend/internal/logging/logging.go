package logging

import (
	"log"
	"sync"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// InitGlobalLogger initializes a global zap logger and redirects the stdlib log output to zap.
func InitGlobalLogger() {
	once.Do(func() {
		l, err := zap.NewProduction()
		if err != nil {
			l, _ = zap.NewDevelopment()
		}
		logger = l
		// redirect stdlib log to zap (uses sugared logger)
		log.SetOutput(zapWriter{logger.Sugar()})
	})
}

// GetLogger returns the initialized zap logger; if not initialized, it initializes now.
func GetLogger() *zap.Logger {
	InitGlobalLogger()
	return logger
}

// zapWriter implements io.Writer and routes writes into a zap SugaredLogger.
type zapWriter struct{ s *zap.SugaredLogger }

func (z zapWriter) Write(p []byte) (n int, err error) {
	if z.s != nil {
		z.s.Info(string(p))
	}
	return len(p), nil
}
