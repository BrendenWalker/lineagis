package semver

import (
	"fmt"
	"regexp"
	"strings"
)

// tagPattern accepts optional v prefix and MAJOR.MINOR.PATCH with optional pre-release and build metadata.
var tagPattern = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// ValidateTag reports whether tag is a valid semantic version per FR-PUB-005.
func ValidateTag(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("tag is required")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid semver tag %q", tag)
	}
	return nil
}
