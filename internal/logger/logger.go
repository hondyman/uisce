package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig = zapcore.EncoderConfig{
		MessageKey:  "msg",
		TimeKey:     "ts",
		LevelKey:    "level",
		EncodeTime:  zapcore.ISO8601TimeEncoder,
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}
	logger, err := config.Build()
	if err != nil {
		// fallback to a production logger if config build fails
		l, _ := zap.NewProduction()
		L = l.Sugar()
		return
	}
	L = logger.Sugar()
}
