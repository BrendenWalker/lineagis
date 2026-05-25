package signing

import "time"

// DefaultTestTimeout is used for cosign CLI operations in tests.
func DefaultTestTimeout() time.Duration {
	return 30 * time.Second
}
