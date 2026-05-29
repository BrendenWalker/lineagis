package api

import "testing"

func TestParseEvalPhase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    EvalPhase
		wantErr bool
	}{
		{"push", EvalPhasePush, false},
		{"verify", EvalPhaseVerify, false},
		{"", "", true},
		{"inspect", "", true},
	}
	for _, tc := range tests {
		got, err := parseEvalPhase(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("parseEvalPhase(%q) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
		}
		if !tc.wantErr && got != tc.want {
			t.Fatalf("parseEvalPhase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRuleAppliesInPhase(t *testing.T) {
	t.Parallel()
	sig := policyRule{ID: "require-signatures"}
	other := policyRule{ID: "trusted-publishers"}
	prov := policyRule{ID: "require-provenance"}
	for _, phase := range []EvalPhase{EvalPhasePush, EvalPhaseVerify} {
		if !ruleAppliesInPhase(sig, phase) {
			t.Fatalf("require-signatures should apply on %s", phase)
		}
		if !ruleAppliesInPhase(other, phase) {
			t.Fatalf("trusted-publishers should apply on %s", phase)
		}
		if !ruleAppliesInPhase(prov, phase) {
			t.Fatalf("require-provenance should apply on %s", phase)
		}
	}
}
