package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrInsufficientCredits = errors.New("créditos insuficientes")
	ErrConflict            = errors.New("recurso já existe")
	ErrTokenExpired        = errors.New("convite expirado")
)
