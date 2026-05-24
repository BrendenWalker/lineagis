package registry

import "errors"

var (
	// ErrNotFound is returned when a requested blob does not exist in the registry.
	ErrNotFound = errors.New("registry: blob not found")
	// ErrBlobTooLarge is returned when blob content exceeds the MVP size limit (ADR-0001).
	ErrBlobTooLarge = errors.New("registry: blob exceeds maximum size")
)
