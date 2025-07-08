package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"

	botcontext "github.com/synaptiq/standup-bot/context"
)

// Handler is a function that processes Lambda requests.
type Handler func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Middleware is a function that wraps a handler.
type Middleware func(Handler) Handler

// Chain chains multiple middleware together.
func Chain(middlewares ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// WithRequestID adds a request ID to the context.
func WithRequestID(botCtx botcontext.BotContext) Middleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			request events.APIGatewayProxyRequest,
		) (events.APIGatewayProxyResponse, error) {
			requestID := request.Headers["X-Request-ID"]
			if requestID == "" {
				requestID = uuid.New().String()
			}

			ctx = botCtx.WithRequestID(ctx, requestID)
			response, err := next(ctx, request)

			// Add request ID to response headers
			if response.Headers == nil {
				response.Headers = make(map[string]string)
			}
			response.Headers["X-Request-ID"] = requestID

			return response, err
		}
	}
}

// WithLogging adds structured logging to requests.
func WithLogging(botCtx botcontext.BotContext) Middleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			request events.APIGatewayProxyRequest,
		) (events.APIGatewayProxyResponse, error) {
			start := time.Now()
			logger := botCtx.Logger()

			// Log request
			logger.Info(ctx, "Request received",
				botcontext.Field{Key: "method", Value: request.HTTPMethod},
				botcontext.Field{Key: "path", Value: request.Path},
				botcontext.Field{Key: "source_ip", Value: request.RequestContext.Identity.SourceIP},
			)

			// Process request
			response, err := next(ctx, request)

			// Log response
			duration := time.Since(start)
			if err != nil {
				logger.Error(ctx, "Request failed", err,
					botcontext.Field{Key: "duration_ms", Value: duration.Milliseconds()},
				)
			} else {
				logger.Info(ctx, "Request completed",
					botcontext.Field{Key: "status_code", Value: response.StatusCode},
					botcontext.Field{Key: "duration_ms", Value: duration.Milliseconds()},
				)
			}

			return response, err
		}
	}
}

// WithTracing adds distributed tracing.
func WithTracing(botCtx botcontext.BotContext) Middleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			request events.APIGatewayProxyRequest,
		) (events.APIGatewayProxyResponse, error) {
			tracer := botCtx.Tracer()
			ctx, done := tracer.StartSpan(ctx, "lambda_handler")
			defer done()

			tracer.AddAnnotation(ctx, "path", request.Path)
			tracer.AddAnnotation(ctx, "method", request.HTTPMethod)

			return next(ctx, request)
		}
	}
}

// WithRecovery recovers from panics and returns 500.
func WithRecovery(botCtx botcontext.BotContext) Middleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			request events.APIGatewayProxyRequest,
		) (response events.APIGatewayProxyResponse, err error) {
			defer func() {
				if r := recover(); r != nil {
					logger := botCtx.Logger()
					logger.Error(ctx, "Panic recovered", fmt.Errorf("%v", r),
						botcontext.Field{Key: "stack", Value: string(debug.Stack())},
					)
					response = InternalServerError("Internal server error")
				}
			}()

			return next(ctx, request)
		}
	}
}

// WithCORS adds CORS headers.
func WithCORS(allowedOrigins []string) Middleware {
	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			request events.APIGatewayProxyRequest,
		) (events.APIGatewayProxyResponse, error) {
			response, err := next(ctx, request)

			// Add CORS headers
			if response.Headers == nil {
				response.Headers = make(map[string]string)
			}

			origin := request.Headers["Origin"]
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				response.Headers["Access-Control-Allow-Origin"] = origin
				response.Headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, OPTIONS"
				response.Headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization, X-Request-ID"
				response.Headers["Access-Control-Max-Age"] = "86400"
			}

			return response, err
		}
	}
}

// ParseBody parses the request body into the given interface.
//
//nolint:gocritic // hugeParam: consistent with handler signatures
func ParseBody(request events.APIGatewayProxyRequest, v interface{}) error {
	if request.Body == "" {
		return fmt.Errorf("empty request body")
	}

	if err := json.Unmarshal([]byte(request.Body), v); err != nil {
		return fmt.Errorf("failed to parse request body: %w", err)
	}

	return nil
}

// ExtractUserID extracts user ID from various sources.
func ExtractUserID(request *events.APIGatewayProxyRequest) string {
	// Check path parameters
	if userID, ok := request.PathParameters["userId"]; ok {
		return userID
	}

	// Check query parameters
	if userID, ok := request.QueryStringParameters["user_id"]; ok {
		return userID
	}

	// Check headers (for authenticated requests)
	if userID := request.Headers["X-User-ID"]; userID != "" {
		return userID
	}

	return ""
}

// StandardMiddleware returns the standard middleware chain.
func StandardMiddleware(botCtx botcontext.BotContext) Middleware {
	return Chain(
		WithRecovery(botCtx),
		WithRequestID(botCtx),
		WithLogging(botCtx),
		WithTracing(botCtx),
	)
}
