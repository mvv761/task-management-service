package middleware

import "context"

type contextKey string

const (
	userIDKey          contextKey = "user_id"
	requestIDKey       contextKey = "request_id"
	requestMetadataKey contextKey = "request_metadata"
)

type requestMetadata struct {
	requestID string
	userID    uint64
	hasUserID bool
}

func contextWithRequestID(ctx context.Context, requestID string, meta *requestMetadata) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	return context.WithValue(ctx, requestMetadataKey, meta)
}

func contextWithUserID(ctx context.Context, userID uint64) context.Context {
	if meta, ok := requestMetadataFromContext(ctx); ok {
		meta.userID = userID
		meta.hasUserID = true
	}

	return context.WithValue(ctx, userIDKey, userID)
}

func requestMetadataFromContext(ctx context.Context) (*requestMetadata, bool) {
	meta, ok := ctx.Value(requestMetadataKey).(*requestMetadata)
	return meta, ok
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok && requestID != ""
}

func UserIDFromContext(ctx context.Context) (uint64, bool) {
	userID, ok := ctx.Value(userIDKey).(uint64)
	return userID, ok
}
