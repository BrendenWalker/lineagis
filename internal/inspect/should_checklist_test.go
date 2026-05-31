package inspect

import (
	"strings"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/apiclient"
)

func TestShouldChecklist_provenanceVerified(t *testing.T) {
	t.Parallel()
	trust := &apiclient.TrustStatus{
		Namespace: "gh/acme/widget",
	}
	trust.Signatures.Status = "valid"
	trust.Attestations.ProvenanceVerified = true
	trust.Attestations.Workflow = "release"

	lines := ShouldChecklist(trust)
	var prov, workflow string
	for _, l := range lines {
		if l.RequirementID == "FR-PROV-006" {
			prov = l.Text
		}
		if l.RequirementID == "FR-PROV-003" {
			workflow = l.Text
		}
	}
	if !strings.Contains(prov, "✓ Provenance verified") {
		t.Fatalf("provenance line = %q", prov)
	}
	if !strings.Contains(workflow, "release") {
		t.Fatalf("workflow line = %q", workflow)
	}
}

func TestShouldChecklist_repositoryNotConfigured(t *testing.T) {
	t.Parallel()
	trust := &apiclient.TrustStatus{
		Namespace: "gh/acme/widget",
	}
	trust.Attestations.Repository = "https://github.com/acme/widget"

	line := repositoryLine(trust)
	if !strings.Contains(line.Text, "repository-ownership not configured") {
		t.Fatalf("text = %q", line.Text)
	}
	if strings.HasPrefix(line.Text, "✓") {
		t.Fatal("must not show pass without repository-ownership configured")
	}
}

func TestShouldChecklist_repositoryVerifiedWhenConfigured(t *testing.T) {
	t.Parallel()
	trust := &apiclient.TrustStatus{
		Namespace:       "gh/acme/widget",
		ConfiguredRules: []string{"repository-ownership"},
	}
	trust.Attestations.Repository = "https://github.com/acme/widget"

	line := repositoryLine(trust)
	if line.Text != "✓ Repository verified" || !line.Pass {
		t.Fatalf("got %+v", line)
	}
}

func TestShouldChecklist_sbomAttached(t *testing.T) {
	t.Parallel()
	trust := &apiclient.TrustStatus{}
	trust.Attestations.SBOM = true

	lines := ShouldChecklist(trust)
	for _, l := range lines {
		if l.RequirementID == "FR-PROV-008" && l.Text == "✓ SBOM attached" && l.Pass {
			return
		}
	}
	t.Fatal("expected SBOM attached line")
}
