package errors

import "errors"

// Errors
var (
	ErrAlreadyRegistered = errors.New("scout already registered")
	ErrNotRegistered     = errors.New("scout not registered")
)
