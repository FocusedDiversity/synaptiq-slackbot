package context

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode"
)

// defaultLogger is a simple logger implementation
type defaultLogger struct{}

func (l *defaultLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log("DEBUG", ctx, msg, fields...)
}

func (l *defaultLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log("INFO", ctx, msg, fields...)
}

func (l *defaultLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log("WARN", ctx, msg, fields...)
}

func (l *defaultLogger) Error(ctx context.Context, msg string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, Field{Key: "error", Value: err.Error()})
	}
	l.log("ERROR", ctx, msg, fields...)
}

func (l *defaultLogger) log(level string, ctx context.Context, msg string, fields ...Field) {
	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields = append([]Field{{Key: "request_id", Value: requestID}}, fields...)
	}

	// Format fields
	fieldStr := ""
	for _, f := range fields {
		// Sanitize field values to prevent log injection
		sanitizedValue := sanitizeForLog(fmt.Sprintf("%v", f.Value))
		fieldStr += fmt.Sprintf(" %s=%s", f.Key, sanitizedValue)
	}

	log.Printf("[%s] %s%s", level, msg, fieldStr)
}

// sanitizeForLog removes control characters and newlines from log values
func sanitizeForLog(value string) string {
	if value == "" {
		return ""
	}

	// Remove newlines, carriage returns, and other control characters
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, value)

	// Trim to reasonable length
	const maxLength = 200
	if len(sanitized) > maxLength {
		sanitized = sanitized[:maxLength] + "..."
	}

	return strings.TrimSpace(sanitized)
}

// noopTracer is a no-op tracer implementation
type noopTracer struct{}

func (t *noopTracer) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *noopTracer) AddAnnotation(ctx context.Context, key string, value interface{}) {
	// No-op
}
