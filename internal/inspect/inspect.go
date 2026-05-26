package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/publish"
	"github.com/BrendenWalker/verity/internal/registry"
)

const reportVersion = 1

// Options configures an inspect run (FR-SIGN-005, FR-DX-002).
type Options struct {
	Namespace string
	Artifact  string
	Ref       string
}

// ChecklistLine is one inspect row for human or JSON output (AC-DX-002, FR-DX-007).
type ChecklistLine struct {
	Text          string
	Must          bool
	Pass          bool
	RequirementID string
	RuleID        string
}

// Report is the machine-readable inspect output (FR-DX-006, AC-DX-005).
type Report struct {
	Version   int           `json:"version"`
	Namespace string        `json:"namespace"`
	Artifact  string        `json:"artifact"`
	Digest    string        `json:"digest"`
	Overall   string        `json:"overall"`
	Checks    []ReportCheck `json:"checks"`
}

// ReportCheck is one row in a JSON inspect report.
type ReportCheck struct {
	Priority      string `json:"priority"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	RequirementID string `json:"requirement_id,omitempty"`
	RuleID        string `json:"rule_id,omitempty"`
}

// Result is the trust checklist outcome for printing and exit codes.
type Result struct {
	Trust     *apiclient.TrustStatus
	MustLines []ChecklistLine
}

// Run resolves ref, fetches API trust status (server-side signature verify), and formats output.
func Run(ctx context.Context, api *apiclient.Client, opts Options) (*Result, error) {
	opts.Namespace = strings.TrimSpace(opts.Namespace)
	opts.Artifact = strings.TrimSpace(opts.Artifact)
	opts.Ref = strings.TrimSpace(opts.Ref)
	if opts.Namespace == "" || opts.Artifact == "" {
		return nil, fmt.Errorf("namespace and artifact are required")
	}
	if opts.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}

	digest, tag, err := resolveRef(opts.Ref)
	if err != nil {
		return nil, err
	}

	trust, err := api.GetTrustStatus(ctx, opts.Namespace, opts.Artifact, digest, tag)
	if err != nil {
		return nil, err
	}

	return &Result{
		Trust:     trust,
		MustLines: MustChecklist(trust),
	}, nil
}

// MustFailed reports whether any Must checklist line failed (FR-DX-005).
func MustFailed(lines []ChecklistLine) bool {
	for _, l := range lines {
		if l.Must && !l.Pass {
			return true
		}
	}
	return false
}

// JSONReport builds the machine-readable inspect document (FR-DX-006).
func JSONReport(result *Result) Report {
	overall := "pass"
	if MustFailed(result.MustLines) {
		overall = "fail"
	}
	checks := make([]ReportCheck, 0, len(result.MustLines))
	for _, line := range result.MustLines {
		priority := "should"
		if line.Must {
			priority = "must"
		}
		status := "pass"
		if !line.Pass {
			status = "fail"
		}
		checks = append(checks, ReportCheck{
			Priority:      priority,
			Status:        status,
			Message:       line.Text,
			RequirementID: line.RequirementID,
			RuleID:        line.RuleID,
		})
	}
	digest := ""
	if result.Trust != nil {
		digest = result.Trust.Digest
	}
	return Report{
		Version:   reportVersion,
		Namespace: result.Trust.Namespace,
		Artifact:  result.Trust.Artifact,
		Digest:    digest,
		Overall:   overall,
		Checks:    checks,
	}
}

// EncodeJSON writes a JSON inspect report to w.
func EncodeJSON(w interface{ Write([]byte) (int, error) }, result *Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(JSONReport(result))
}

// MustChecklist builds MVP Must output lines from API trust status (AC-DX-002).
func MustChecklist(trust *apiclient.TrustStatus) []ChecklistLine {
	lines := []ChecklistLine{signatureLine(trust.Signatures.Status, trust.Policy.Reasons)}
	lines = append(lines, policyMustLines(trust)...)
	return lines
}

func policyMustLines(trust *apiclient.TrustStatus) []ChecklistLine {
	if trust.Policy.Status != "fail" {
		return nil
	}
	if trust.Signatures.Status != "valid" {
		return nil
	}
	var out []ChecklistLine
	for _, r := range trust.Policy.Reasons {
		reqID := policyRequirementID(r.Rule)
		out = append(out, ChecklistLine{
			Text:          failLine(r.Message, reqID, r.Rule, r.Message),
			Must:          true,
			Pass:          false,
			RequirementID: reqID,
			RuleID:        r.Rule,
		})
	}
	return out
}

func policyRequirementID(rule string) string {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "require-signatures", "require-signature":
		return "FR-POL-005"
	default:
		return "FR-POL-004"
	}
}

func signatureLine(status string, policyReasons []apiclient.PolicyReason) ChecklistLine {
	ruleID := policyRuleForSignatures(policyReasons)
	switch status {
	case "valid":
		return ChecklistLine{
			Text:          "✓ Signed by GitHub Actions",
			Must:          true,
			Pass:          true,
			RequirementID: "FR-SIGN-005",
		}
	case "missing":
		return ChecklistLine{
			Text: failLine(
				"Signature missing",
				"FR-SIGN-005",
				ruleID,
				"attach a Sigstore bundle (e.g. publish from GitHub Actions with OIDC)",
			),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-SIGN-005",
			RuleID:        ruleID,
		}
	case "invalid":
		return ChecklistLine{
			Text: failLine(
				"Signature invalid",
				"FR-SIGN-005",
				ruleID,
				"re-publish with a valid Sigstore signature for this digest",
			),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-SIGN-005",
			RuleID:        ruleID,
		}
	default:
		return ChecklistLine{
			Text: failLine(
				fmt.Sprintf("Signature status unknown (%s)", status),
				"FR-SIGN-005",
				ruleID,
				"confirm trust status with the Verity API or re-publish",
			),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-SIGN-005",
			RuleID:        ruleID,
		}
	}
}

func policyRuleForSignatures(reasons []apiclient.PolicyReason) string {
	for _, r := range reasons {
		switch strings.ToLower(strings.TrimSpace(r.Rule)) {
		case "require-signatures", "require-signature":
			return r.Rule
		}
	}
	return ""
}

func failLine(check, requirementID, ruleID, hint string) string {
	refs := requirementID
	if ruleID != "" {
		refs = requirementID + ", rule " + ruleID
	}
	return fmt.Sprintf("✗ %s (%s): %s", check, refs, hint)
}

func resolveRef(ref string) (digest, tag string, err error) {
	if strings.HasPrefix(ref, "sha256:") {
		return ref, "", nil
	}
	if _, err := os.Stat(ref); err == nil {
		layers, publishRoot, err := publish.CollectFiles(ref)
		if err != nil {
			return "", "", err
		}
		_, h, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{PublishRoot: publishRoot})
		if err != nil {
			return "", "", err
		}
		return h.String(), "", nil
	}
	return "", ref, nil
}
