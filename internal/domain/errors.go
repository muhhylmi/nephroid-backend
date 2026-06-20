package domain

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrConflict      = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input data")
	ErrInternal      = errors.New("internal server error")
	ErrUnauthorized  = errors.New("unauthorized access")
)
