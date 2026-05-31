package publish

import "github.com/BrendenWalker/lineagis/internal/signing"

// SkipSignFromEnv reports whether signing should be skipped (local dev).
func SkipSignFromEnv() bool {
	return signing.SkipSignFromEnv()
}
