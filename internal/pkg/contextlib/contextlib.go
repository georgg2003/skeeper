// Package contextlib stores request metadata and the authenticated user id on [context.Context]
// for handlers and logging.
package contextlib

import (
	"context"
)

type userKey struct{}

var uk = userKey{}

func SetUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, uk, userID)
}

func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(uk).(int64)
	return userID, ok
}

type RequestInfo struct {
	RequestID string
	Host      string
	RemoteIP  string
	Method    string
	Path      string
	UserAgent string
}

type reqInfoKey struct{}

var rik = reqInfoKey{}

func SetRequestInfo(ctx context.Context, requestInfo RequestInfo) context.Context {
	return context.WithValue(ctx, rik, requestInfo)
}

func GetRequestInfo(ctx context.Context) (RequestInfo, bool) {
	requestInfo, ok := ctx.Value(rik).(RequestInfo)
	return requestInfo, ok
}
