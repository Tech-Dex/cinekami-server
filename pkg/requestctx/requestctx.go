package requestctx

import "context"

type ctxKey string

const correlationIDKey ctxKey = "correlation_id"

// WithCorrelationID returns a new context with the provided correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID fetches the correlation ID from the context, if any.
func CorrelationID(ctx context.Context) string {
	v := ctx.Value(correlationIDKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
