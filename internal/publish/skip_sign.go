package publish

import "github.com/BrendenWalker/verity/internal/signing"

// SkipSignFromEnv reports whether signing should be skipped (local dev).
func SkipSignFromEnv() bool {
	return signing.SkipSignFromEnv()
}
