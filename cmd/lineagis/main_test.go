package main

import "testing"

func TestRunVersion(t *testing.T) {
	if got := run([]string{"--version"}); got != 0 {
		t.Fatalf("run(--version) = %d, want 0", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if got := run([]string{"publish"}); got != 1 {
		t.Fatalf("run(publish) = %d, want 1", got)
	}
}

func TestRunTraceFormatFlag(t *testing.T) {
	// Requires graph; only verify flag parsing does not error on --format.
	if got := run([]string{"trace", "artifact:sha256:missing", "--format", "json"}); got == 2 {
		t.Fatal("trace --format should not fail with usage (exit 2)")
	}
}
