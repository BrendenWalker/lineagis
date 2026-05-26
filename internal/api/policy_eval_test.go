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
	if !ruleAppliesInPhase(sig, EvalPhasePush) {
		t.Fatal("require-signatures should apply on push")
	}
	if !ruleAppliesInPhase(sig, EvalPhaseVerify) {
		t.Fatal("require-signatures should apply on verify")
	}
	if ruleAppliesInPhase(other, EvalPhasePush) {
		t.Fatal("trusted-publishers should not apply on push in MVP")
	}
	if !ruleAppliesInPhase(other, EvalPhaseVerify) {
		t.Fatal("trusted-publishers should apply on verify")
	}
}
