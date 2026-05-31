package main

import "testing"

func TestRunVersion(t *testing.T) {
	t.Parallel()
	if got := run([]string{"--version"}); got != 0 {
		t.Fatalf("run(--version) = %d, want 0", got)
	}
}

func TestRunPublishMissingFlags(t *testing.T) {
	t.Setenv("LINEAGIS_TOKEN", "tok")
	if got := run([]string{"publish", ".", "--namespace", "ns", "--artifact", "a"}); got != 1 {
		t.Fatalf("run(publish) = %d, want 1 (empty dir)", got)
	}
}

func TestRunNoArgs(t *testing.T) {
	t.Parallel()
	if got := run(nil); got != 1 {
		t.Fatalf("run() = %d, want 1", got)
	}
}

func TestRunInspectMissingFlags(t *testing.T) {
	t.Parallel()
	if got := run([]string{"inspect", "sha256:abc"}); got != 1 {
		t.Fatalf("run(inspect) = %d, want 1", got)
	}
}
