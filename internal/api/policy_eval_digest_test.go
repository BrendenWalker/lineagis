package api

import "testing"

func TestRuleRequireDigestOnVerify_phase(t *testing.T) {
	t.Parallel()
	rule := policyRule{Type: "require-digest-on-verify"}
	if ruleAppliesInPhase(rule, EvalPhasePush) {
		t.Fatal("must not run at push time")
	}
	if !ruleAppliesInPhase(rule, EvalPhaseVerify) {
		t.Fatal("must run at verify time")
	}
	if !ruleRequireDigestOnVerify(rule) {
		t.Fatal("rule matcher")
	}
}
