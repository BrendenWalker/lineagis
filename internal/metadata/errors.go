package metadata

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("metadata: not found")
	// ErrDigestWrongArtifact is returned when a digest does not belong to the target artifact.
	ErrDigestWrongArtifact = errors.New("metadata: digest does not belong to artifact")
)
