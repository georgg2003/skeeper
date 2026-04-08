package usecase

import "github.com/georgg2003/skeeper/pkg/errors"

// ErrUnauthenticated means the JWT interceptor never set a user id on the context.
var ErrUnauthenticated = errors.New("unauthenticated")
