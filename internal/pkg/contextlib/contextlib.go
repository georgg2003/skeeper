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

func MustGetUserID(ctx context.Context) int64 {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("failed to get user id from context")
	}
	return userID
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
