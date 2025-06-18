package logger

import (
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Create(loglevel string) *zap.Logger {
	loglevel = strings.ToUpper(loglevel)

	if loglevel == "" || loglevel == "PROD" {
		os.Setenv("NUBO_LOG", "PROD")
		return zap.NewNop()
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	switch loglevel {
	default:
		os.Setenv("NUBO_LOG", "PROD")
		return zap.NewNop()
	case "DEBUG":
		os.Setenv("NUBO_LOG", "DEBUG")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "INFO":
		os.Setenv("NUBO_LOG", "INFO")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "WARN":
		os.Setenv("NUBO_LOG", "WARN")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "ERROR":
		os.Setenv("NUBO_LOG", "ERROR")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "DPANIC":
		os.Setenv("NUBO_LOG", "DPANIC")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DPanicLevel)
	case "PANIC":
		os.Setenv("NUBO_LOG", "PANIC")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	case "FATAL":
		os.Setenv("NUBO_LOG", "FATAL")
		cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	}

	logger, err := cfg.Build()
	if err != nil {
		log.Fatal(err)
	}

	return logger
}
