package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NOOPLogger use for unit testing
var NOOPLogger = zap.NewNop().Sugar()

func NewAppLogger() (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.EncoderConfig.CallerKey = "func"
	cfg.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

func Sync(l *zap.SugaredLogger) {
	if err := l.Sync(); err != nil {
		l.Error("cannot sync logger: ", err)
	}
}
