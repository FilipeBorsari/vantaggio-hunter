package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrInsufficientCredits = errors.New("créditos insuficientes")
)
