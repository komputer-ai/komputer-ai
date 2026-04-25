// komputer-api/logger.go
package main

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// Logger is the package-level structured logger. Initialized by InitLogger.
var Logger *zap.SugaredLogger

// InitLogger configures the global Logger based on env vars:
//   - LOG_LEVEL: debug|info|warn|error (default info)
//   - LOG_FORMAT: json|text (default: auto — text when stdout is a TTY, json otherwise)
func InitLogger() {
	level := zapcore.InfoLevel
	if l, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch strings.ToLower(l) {
		case "debug":
			level = zapcore.DebugLevel
		case "warn", "warning":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		}
	}

	useJSON := !term.IsTerminal(int(os.Stdout.Fd()))
	if f, ok := os.LookupEnv("LOG_FORMAT"); ok {
		switch strings.ToLower(f) {
		case "json":
			useJSON = true
		case "text":
			useJSON = false
		}
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.MessageKey = "message"
	encoderCfg.LevelKey = "level"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if useJSON {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)
	logger := zap.New(core).With(zap.String("component", "komputer-api"))
	Logger = logger.Sugar()
}
