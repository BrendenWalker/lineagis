package signing

import "testing"

func TestGitHubPublisherFromBundle_testKey(t *testing.T) {
	t.Parallel()
	bundle, _, err := SignManifestForTest([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	_, ok := GitHubPublisherFromBundle(bundle)
	if ok {
		t.Log("github identity present in test cert")
	}
}
