package db

import (
	"github.com/georgg2003/skeeper/pkg/errors"
)

var ErrUserExists = errors.New("user already exists")
var ErrInvalidToken = errors.New("invalid token")
var ErrUserNotExist = errors.New("user does not exist")
