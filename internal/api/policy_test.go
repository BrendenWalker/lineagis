package api

import (
	"encoding/json"
	"testing"
)

func TestPolicyRequiresSignatures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want bool
	}{
		{"empty", `{}`, false},
		{"no rules", `{"rules":[]}`, false},
		{"id plural", `{"rules":[{"id":"require-signatures"}]}`, true},
		{"id singular", `{"rules":[{"id":"require-signature"}]}`, true},
		{"type plural", `{"rules":[{"type":"require-signatures"}]}`, true},
		{"type singular", `{"rules":[{"type":"require-signature"}]}`, true},
		{"other rule", `{"rules":[{"id":"trusted-publishers"}]}`, false},
		{"malformed", `{`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := policyRequiresSignatures(json.RawMessage(tc.doc))
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPolicyFailure_message(t *testing.T) {
	e := PolicyFailure{Rule: "require-signatures", Hint: "attach bundle"}
	if e.Error() != "require-signatures: attach bundle" {
		t.Fatalf("Error() = %q", e.Error())
	}
}
