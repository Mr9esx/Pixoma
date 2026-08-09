package domain

import "errors"

var (
	ErrNotFound   = errors.New("tg menu not found")
	ErrValidation = errors.New("tg menu validation failed")
)
