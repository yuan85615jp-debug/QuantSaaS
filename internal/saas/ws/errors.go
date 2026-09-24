package ws

import "errors"

var (
	ErrNoAgent      = errors.New("no online agent for instance")
	ErrUnauthorized = errors.New("unauthorized")
)
