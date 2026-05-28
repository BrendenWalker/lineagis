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

// TrustHeader is printed before human checklist lines (v0.1 honesty: server-side verify).
const TrustHeader = "Trust verified by Verity API (server-side Sigstore checks)"

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
	Trust       *apiclient.TrustStatus
	MustLines   []ChecklistLine
	ShouldLines []ChecklistLine
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
		Trust:       trust,
		MustLines:   MustChecklist(trust),
		ShouldLines: ShouldChecklist(trust),
	}, nil
}

// HumanLines returns printable inspect rows including the trust header.
func HumanLines(result *Result) []string {
	if result == nil {
		return []string{TrustHeader}
	}
	lines := []string{TrustHeader}
	for _, l := range result.MustLines {
		lines = append(lines, l.Text)
	}
	for _, l := range result.ShouldLines {
		lines = append(lines, l.Text)
	}
	return lines
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
	all := append(append([]ChecklistLine{}, result.MustLines...), result.ShouldLines...)
	checks := make([]ReportCheck, 0, len(all))
	for _, line := range all {
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

// ShouldChecklist builds Phase 2 Should lines (FR-PROV-006, FR-PROV-008, FR-PROV-012, FR-PROV-013).
func ShouldChecklist(trust *apiclient.TrustStatus) []ChecklistLine {
	if trust == nil {
		return nil
	}
	var lines []ChecklistLine
	lines = append(lines, repositoryLine(trust))
	lines = append(lines, maintainerLine(trust))
	lines = append(lines, sbomLine(trust))
	lines = append(lines, provenanceLine(trust))
	lines = append(lines, workflowLine(trust))
	return lines
}

func repositoryLine(trust *apiclient.TrustStatus) ChecklistLine {
	if trust.Attestations.Repository == "" {
		return ChecklistLine{
			Text:          "⚠ Repository not verified (no provenance repository)",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-012",
		}
	}
	expected := namespaceRepo(trust.Namespace)
	if expected != "" && !strings.EqualFold(repoFromURI(trust.Attestations.Repository), expected) {
		return ChecklistLine{
			Text:          fmt.Sprintf("✗ Repository mismatch (%s)", trust.Attestations.Repository),
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-012",
		}
	}
	return ChecklistLine{
		Text:          "✓ Repository verified",
		Must:          false,
		Pass:          true,
		RequirementID: "FR-PROV-012",
	}
}

func maintainerLine(trust *apiclient.TrustStatus) ChecklistLine {
	for _, r := range trust.Policy.Reasons {
		if strings.EqualFold(r.Rule, "trusted-publishers") {
			return ChecklistLine{
				Text:          failLine("Maintainer not verified", "FR-PROV-013", r.Rule, r.Message),
				Must:          false,
				Pass:          false,
				RequirementID: "FR-PROV-013",
				RuleID:        r.Rule,
			}
		}
	}
	if trust.Signatures.Status != "valid" {
		return ChecklistLine{
			Text:          "⚠ Maintainer not verified (signature missing or invalid)",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-013",
		}
	}
	return ChecklistLine{
		Text:          "✓ Maintainer verified",
		Must:          false,
		Pass:          true,
		RequirementID: "FR-PROV-013",
	}
}

func sbomLine(trust *apiclient.TrustStatus) ChecklistLine {
	if trust.Attestations.SBOM {
		return ChecklistLine{
			Text:          "✓ SBOM attached",
			Must:          false,
			Pass:          true,
			RequirementID: "FR-PROV-008",
		}
	}
	return ChecklistLine{
		Text:          "⚠ SBOM not attached",
		Must:          false,
		Pass:          false,
		RequirementID: "FR-PROV-008",
	}
}

func provenanceLine(trust *apiclient.TrustStatus) ChecklistLine {
	if trust.Attestations.ProvenanceVerified {
		return ChecklistLine{
			Text:          "✓ Provenance verified",
			Must:          false,
			Pass:          true,
			RequirementID: "FR-PROV-006",
		}
	}
	if trust.Attestations.Provenance {
		return ChecklistLine{
			Text:          "✗ Provenance invalid or signature failed",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-006",
		}
	}
	return ChecklistLine{
		Text:          "⚠ Provenance not attached",
		Must:          false,
		Pass:          false,
		RequirementID: "FR-PROV-006",
	}
}

func workflowLine(trust *apiclient.TrustStatus) ChecklistLine {
	if trust.Attestations.Workflow == "" {
		return ChecklistLine{
			Text:          "⚠ GitHub Actions workflow identity unavailable",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-003",
		}
	}
	msg := fmt.Sprintf("✓ Published via workflow %s", trust.Attestations.Workflow)
	if trust.Attestations.WorkflowRef != "" {
		msg += " (" + trust.Attestations.WorkflowRef + ")"
	}
	if trust.Attestations.RunID != "" {
		msg += " run " + trust.Attestations.RunID
	}
	return ChecklistLine{
		Text:          msg,
		Must:          false,
		Pass:          true,
		RequirementID: "FR-PROV-003",
	}
}

func namespaceRepo(ns string) string {
	const prefix = "gh/"
	if strings.HasPrefix(ns, prefix) {
		return strings.TrimPrefix(ns, prefix)
	}
	return ""
}

func repoFromURI(uri string) string {
	uri = strings.TrimSuffix(strings.TrimSpace(uri), "/")
	if strings.HasPrefix(uri, "https://github.com/") {
		return strings.TrimPrefix(uri, "https://github.com/")
	}
	return uri
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
