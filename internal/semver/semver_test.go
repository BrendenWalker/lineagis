package semver

import "testing"

func TestValidateTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tag   string
		valid bool
	}{
		{"v1.0.0", true},
		{"1.0.0", true},
		{"v2.3.4-beta.1", true},
		{"v0.0.0", true},
		{"latest", false},
		{"", false},
		{"v1", false},
	}
	for _, tc := range tests {
		err := ValidateTag(tc.tag)
		if tc.valid && err != nil {
			t.Errorf("ValidateTag(%q) = %v, want nil", tc.tag, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("ValidateTag(%q) = nil, want error", tc.tag)
		}
	}
}
