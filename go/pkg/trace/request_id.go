package trace

import (
	"context"
	"github.com/google/uuid"
)

type contextKey string

// RequestIDKey is the context / gin key storing the request identifier.
const RequestIDKey contextKey = "request_id"

// WithRequestID adds the given request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// FromContext retrieves the request ID from context, generating one if missing.
func FromContext(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDKey).(string); ok && val != "" {
		return val
	}
	return uuid.NewString()
}

// RandomID generates a new request identifier.
func RandomID() string {
	return uuid.NewString()
}
