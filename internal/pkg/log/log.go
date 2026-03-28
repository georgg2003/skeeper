package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
)

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		return h.Handler.Handle(ctx, r)
	}

	reqInfo, ok := contextlib.GetRequestInfo(ctx)
	if !ok {
		return h.Handler.Handle(ctx, r)
	}
	userIDStr := ""
	userID, ok := contextlib.GetUserID(ctx)
	if ok {
		userIDStr = fmt.Sprint(userID)
	}
	r.AddAttrs(
		slog.String("request_id", reqInfo.RequestID),
		slog.String("remote_ip", reqInfo.RemoteIP),
		slog.String("method", reqInfo.Method),
		slog.String("path", reqInfo.Path),
		slog.String("user_agent", reqInfo.UserAgent),
		slog.String("user_id", userIDStr),
	)

	return h.Handler.Handle(ctx, r)
}

func New() *slog.Logger {
	handler := &ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)}
	return slog.New(handler)
}
