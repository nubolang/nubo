package logger

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Create(loglevel string) *zap.Logger {
	if loglevel == "" {
		loglevel = config.Current.Logging.Level
	}

	lvl := strings.ToUpper(loglevel)
	var level zapcore.Level

	switch lvl {
	case "PROD", "PRODUCTION":
		return zap.NewNop()
	case "DEBUG":
		level = zapcore.DebugLevel
	case "INFO":
		level = zapcore.InfoLevel
	case "WARN":
		level = zapcore.WarnLevel
	case "ERROR":
		level = zapcore.ErrorLevel
	case "DPANIC":
		level = zapcore.DPanicLevel
	case "PANIC":
		level = zapcore.PanicLevel
	case "FATAL":
		level = zapcore.FatalLevel
	default:
		level = zapcore.ErrorLevel // fallback
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var cores []zapcore.Core

	if config.Current.Logging.Loggers.Console.Use {
		enc := zapcore.NewConsoleEncoder(encCfg)
		cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), level))
	}

	if config.Current.Logging.Loggers.File.Use {
		path := config.Current.Logging.Loggers.File.Path
		folder := filepath.Dir(path)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			log.Printf("failed to create log folder: %v", err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			enc := zapcore.NewJSONEncoder(encCfg)
			cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(f), level))
		} else {
			log.Printf("failed to open log file: %v", err)
		}
	}

	if len(cores) == 0 {
		return zap.NewNop()
	}

	return zap.New(zapcore.NewTee(cores...))
}
