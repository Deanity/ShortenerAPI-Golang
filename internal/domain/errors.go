package domain

import "errors"

var (
	ErrLinkNotFound      = errors.New("link not found")
	ErrSlugTaken         = errors.New("slug already taken")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken      = errors.New("invalid token")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrLinkExpired       = errors.New("link has expired")
	ErrClickLimitReached = errors.New("click limit has been reached")
	ErrPasswordRequired  = errors.New("password required for this link")
	ErrInvalidPassword   = errors.New("invalid password for link")
)
