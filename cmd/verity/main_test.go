package main

import "testing"

func TestRunVersion(t *testing.T) {
	t.Parallel()
	if got := run([]string{"--version"}); got != 0 {
		t.Fatalf("run(--version) = %d, want 0", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()
	if got := run([]string{"publish"}); got != 1 {
		t.Fatalf("run(publish) = %d, want 1", got)
	}
}

func TestRunNoArgs(t *testing.T) {
	t.Parallel()
	if got := run(nil); got != 1 {
		t.Fatalf("run() = %d, want 1", got)
	}
}
