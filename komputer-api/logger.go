// komputer-api/logger.go
package main

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// errorCaptureWriter wraps gin's ResponseWriter to capture the response body
// when the status code indicates an error, so the access log can surface it.
type errorCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *errorCaptureWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// accessLogMiddleware emits one structured Debug log per HTTP request, with
// method, path, status, latency, and client IP. At INFO and above this is
// silent — Prometheus already records the same data. Set LOG_LEVEL=debug to
// see access lines (replaces gin's default `[GIN] ...` line printer).
//
// For 4xx/5xx responses, the response body is captured and included as `error`
// so failures are diagnosable from logs (otherwise the error JSON only reaches the client).
//
// Skips noisy paths (metrics, health probes) so debug mode stays useful.
func accessLogMiddleware() gin.HandlerFunc {
	skip := map[string]struct{}{
		"/api/metrics":   {},
		"/agent/metrics": {},
		"/healthz":       {},
		"/readyz":        {},
	}
	return func(c *gin.Context) {
		start := time.Now()
		capture := &errorCaptureWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = capture
		c.Next()
		path := c.Request.URL.Path
		if _, isSkipped := skip[path]; isSkipped {
			return
		}
		status := c.Writer.Status()
		fields := []interface{}{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if status >= 400 && capture.body.Len() > 0 {
			body := capture.body.String()
			if len(body) > 1024 {
				body = body[:1024] + "...(truncated)"
			}
			fields = append(fields, "error", body)
			Logger.Warnw("http request error", fields...)
			return
		}
		Logger.Debugw("http request", fields...)
	}
}
