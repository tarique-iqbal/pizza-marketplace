package auth

import "errors"

var ErrRefreshTokenInvalid = errors.New("refresh token not found or expired")
