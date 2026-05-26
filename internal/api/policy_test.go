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

func TestValidatePolicyDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		doc     string
		wantErr bool
	}{
		{name: "empty object", doc: `{}`, wantErr: false},
		{name: "empty rules", doc: `{"rules":[]}`, wantErr: false},
		{name: "valid rule by id", doc: `{"rules":[{"id":"require-signatures"}]}`, wantErr: false},
		{name: "valid rule by type", doc: `{"rules":[{"type":"trusted-publishers"}]}`, wantErr: false},
		{name: "invalid root type", doc: `[]`, wantErr: true},
		{name: "unknown root field", doc: `{"foo":"bar"}`, wantErr: true},
		{name: "rule missing id and type", doc: `{"rules":[{}]}`, wantErr: true},
		{name: "unknown rule field", doc: `{"rules":[{"id":"x","mode":"strict"}]}`, wantErr: true},
		{name: "multiple json values", doc: `{} {}`, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePolicyDocument(json.RawMessage(tc.doc))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
