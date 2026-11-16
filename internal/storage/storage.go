package storage

import "errors"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExits    = errors.New("url exists")
)
