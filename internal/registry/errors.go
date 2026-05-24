package registry

import "errors"

var (
	// ErrNotFound is returned when a requested blob or manifest does not exist in the registry.
	ErrNotFound = errors.New("registry: not found")
	// ErrBlobTooLarge is returned when blob content exceeds the MVP size limit (ADR-0001).
	ErrBlobTooLarge = errors.New("registry: blob exceeds maximum size")
	// ErrTooManyLayers is returned when a manifest exceeds the layer count limit (ADR-0001).
	ErrTooManyLayers = errors.New("registry: layer count exceeds maximum")
	// ErrReleaseTooLarge is returned when total release size exceeds the MVP limit (ADR-0001).
	ErrReleaseTooLarge = errors.New("registry: total release size exceeds maximum")
)
