package handler

import (
	"log/slog"
	"net/http"

	appmiddleware "task-management-service/internal/http/middleware"
)

func logHandlerError(r *http.Request, msg string, err error, attrs ...any) {
	logAttrs := make([]any, 0, len(attrs)+6)

	if requestID, ok := appmiddleware.RequestIDFromContext(r.Context()); ok {
		logAttrs = append(logAttrs, "request_id", requestID)
	}

	if userID, ok := appmiddleware.UserIDFromContext(r.Context()); ok {
		logAttrs = append(logAttrs, "user_id", userID)
	}

	logAttrs = append(logAttrs, "error", err)
	logAttrs = append(logAttrs, attrs...)

	slog.Default().Error(msg, logAttrs...)
}
