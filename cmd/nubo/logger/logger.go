package logger

import (
	"log"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Create(loglevel string) *zap.Logger {
	loglevel = strings.ToUpper(loglevel)

	if loglevel == "" || loglevel == "PROD" {
		return zap.NewNop()
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	switch loglevel {
	case "DEBUG":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "INFO":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "WARN":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "ERROR":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "DPANIC":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DPanicLevel)
	case "PANIC":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	case "FATAL":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	}

	logger, err := cfg.Build()
	if err != nil {
		log.Fatal(err)
	}

	return logger
}
