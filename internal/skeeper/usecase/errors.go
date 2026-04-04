package usecase

import "github.com/georgg2003/skeeper/pkg/errors"

// ErrUnauthenticated is returned when the request context has no authenticated user id.
var ErrUnauthenticated = errors.New("unauthenticated")
