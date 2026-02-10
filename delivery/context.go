package delivery

import "context"

type contextKey string

const replyTargetContextKey contextKey = "delivery.reply_to"

// WithTarget stores a normalized reply target on context.
func WithTarget(ctx context.Context, target *Target) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := Normalize(target)
	if normalized == nil {
		return ctx
	}
	return context.WithValue(ctx, replyTargetContextKey, normalized)
}

// FromContext returns the reply target previously attached to context.
func FromContext(ctx context.Context) *Target {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(replyTargetContextKey)
	target, ok := v.(*Target)
	if !ok {
		return nil
	}
	return Normalize(target)
}
