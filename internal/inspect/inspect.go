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

// TrustHeaderAPI notes server-side policy evaluation.
const TrustHeaderAPI = "Policy evaluated by Verity API"

// TrustHeaderLocal notes local Sigstore verification.
const TrustHeaderLocal = "Signature verified locally (Sigstore/Rekor)"

// Options configures an inspect run (FR-SIGN-005, FR-DX-002).
type Options struct {
	Namespace     string
	Artifact      string
	Ref           string
	LocalVerify   bool
	RegistryURL   string
	RequireDigest bool
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
	Version    int           `json:"version"`
	Namespace  string        `json:"namespace"`
	Artifact   string        `json:"artifact"`
	Digest     string        `json:"digest"`
	Overall    string        `json:"overall"`
	TagWarning string        `json:"tag_warning,omitempty"`
	Checks     []ReportCheck `json:"checks"`
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
	LocalVerify *LocalVerifyResult
	ResolvedTag string
	TagWarning  string
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
	if opts.RequireDigest && tag != "" {
		return nil, fmt.Errorf("ref must be a sha256:… digest when --require-digest is set")
	}

	trust, err := api.GetTrustStatus(ctx, opts.Namespace, opts.Artifact, digest, tag)
	if err != nil {
		return nil, err
	}
	if trust.Digest != "" {
		digest = trust.Digest
	}

	var local *LocalVerifyResult
	if opts.LocalVerify && opts.RegistryURL != "" {
		reg, err := registry.New(opts.RegistryURL)
		if err != nil {
			return nil, fmt.Errorf("registry client: %w", err)
		}
		local, err = VerifyLocally(ctx, reg, api, opts.Namespace, opts.Artifact, digest)
		if err != nil {
			return nil, err
		}
	}

	tagWarning := ""
	if tag != "" {
		tagWarning = fmt.Sprintf("warning: tag %q is mutable (resolved to %s); prefer @sha256:… in CI", tag, digest)
	}

	return &Result{
		Trust:       trust,
		LocalVerify: local,
		ResolvedTag: tag,
		TagWarning:  tagWarning,
		MustLines:   MustChecklist(trust, local),
		ShouldLines: ShouldChecklist(trust),
	}, nil
}

// HumanLines returns printable inspect rows including trust headers.
func HumanLines(result *Result) []string {
	if result == nil {
		return []string{TrustHeaderAPI}
	}
	var lines []string
	if result.LocalVerify != nil {
		lines = append(lines, TrustHeaderLocal)
	} else {
		lines = append(lines, TrustHeaderAPI+" (server-side signature checks)")
	}
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
		Version:    reportVersion,
		Namespace:  result.Trust.Namespace,
		Artifact:   result.Trust.Artifact,
		Digest:     digest,
		Overall:    overall,
		TagWarning: result.TagWarning,
		Checks:     checks,
	}
}

// EncodeJSON writes a JSON inspect report to w.
func EncodeJSON(w interface{ Write([]byte) (int, error) }, result *Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(JSONReport(result))
}

// MustChecklist builds MVP Must output lines from API trust status and optional local verify.
func MustChecklist(trust *apiclient.TrustStatus, local *LocalVerifyResult) []ChecklistLine {
	var lines []ChecklistLine
	if local != nil {
		lines = append(lines, localVerifyLine(local))
	}
	lines = append(lines, signatureLine(trust.Signatures.Status, trust.Policy.Reasons, trust.Signer))
	lines = append(lines, policyMustLines(trust)...)
	if ruleConfigured(trust, "require-provenance") {
		lines = append(lines, provenanceMustLine(trust))
	}
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
	if !ruleConfigured(trust, "repository-ownership") {
		return ChecklistLine{
			Text:          "— Repository verified (repository-ownership not configured)",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-012",
		}
	}
	for _, r := range trust.Policy.Reasons {
		if strings.EqualFold(r.Rule, "repository-ownership") {
			return ChecklistLine{
				Text:          failLine("Repository not verified", "FR-PROV-012", r.Rule, r.Message),
				Must:          false,
				Pass:          false,
				RequirementID: "FR-PROV-012",
				RuleID:        r.Rule,
			}
		}
	}
	if trust.Attestations.Repository == "" {
		return ChecklistLine{
			Text:          "✗ Repository not verified (no provenance repository)",
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-012",
			RuleID:        "repository-ownership",
		}
	}
	expected := namespaceRepo(trust.Namespace)
	if expected != "" && !strings.EqualFold(repoFromURI(trust.Attestations.Repository), expected) {
		return ChecklistLine{
			Text:          fmt.Sprintf("✗ Repository mismatch (%s)", trust.Attestations.Repository),
			Must:          false,
			Pass:          false,
			RequirementID: "FR-PROV-012",
			RuleID:        "repository-ownership",
		}
	}
	return ChecklistLine{
		Text:          "✓ Repository verified",
		Must:          false,
		Pass:          true,
		RequirementID: "FR-PROV-012",
		RuleID:        "repository-ownership",
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
	if ruleConfigured(trust, "require-provenance") {
		return ChecklistLine{}
	}
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

func ruleConfigured(trust *apiclient.TrustStatus, rule string) bool {
	if trust == nil {
		return false
	}
	rule = strings.ToLower(strings.TrimSpace(rule))
	for _, r := range trust.ConfiguredRules {
		if strings.EqualFold(strings.TrimSpace(r), rule) {
			return true
		}
	}
	return false
}

func localVerifyLine(local *LocalVerifyResult) ChecklistLine {
	switch local.Status {
	case "valid":
		msg := "✓ Local Sigstore signature valid"
		if local.Signer.Repository != "" {
			msg = fmt.Sprintf("✓ Signed by %s (%s)", local.Signer.Repository, local.Signer.Workflow)
			if local.Signer.Ref != "" {
				msg += " ref " + local.Signer.Ref
			}
		}
		return ChecklistLine{Text: msg, Must: true, Pass: true, RequirementID: "FR-SIGN-005"}
	case "missing":
		return ChecklistLine{
			Text:          failLine("Local signature missing", "FR-SIGN-005", "", "no Sigstore bundle found for digest"),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-SIGN-005",
		}
	default:
		return ChecklistLine{
			Text:          failLine("Local signature invalid", "FR-SIGN-005", "", "cosign verify failed for all bundles"),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-SIGN-005",
		}
	}
}

func provenanceMustLine(trust *apiclient.TrustStatus) ChecklistLine {
	if trust.Attestations.ProvenanceVerified {
		return ChecklistLine{
			Text:          "✓ Provenance verified (require-provenance)",
			Must:          true,
			Pass:          true,
			RequirementID: "FR-PROV-011",
			RuleID:        "require-provenance",
		}
	}
	for _, r := range trust.Policy.Reasons {
		if strings.EqualFold(r.Rule, "require-provenance") {
			return ChecklistLine{
				Text:          failLine("Provenance required", "FR-PROV-011", r.Rule, r.Message),
				Must:          true,
				Pass:          false,
				RequirementID: "FR-PROV-011",
				RuleID:        r.Rule,
			}
		}
	}
	if trust.Attestations.Provenance {
		return ChecklistLine{
			Text:          failLine("Provenance invalid", "FR-PROV-011", "require-provenance", "provenance signature verification failed"),
			Must:          true,
			Pass:          false,
			RequirementID: "FR-PROV-011",
			RuleID:        "require-provenance",
		}
	}
	return ChecklistLine{
		Text:          failLine("Provenance missing", "FR-PROV-011", "require-provenance", "attach SLSA provenance during publish"),
		Must:          true,
		Pass:          false,
		RequirementID: "FR-PROV-011",
		RuleID:        "require-provenance",
	}
}

func policyMustLines(trust *apiclient.TrustStatus) []ChecklistLine {
	if trust == nil {
		return nil
	}
	if trust.Policy.Status != "fail" {
		return nil
	}
	if trust.Signatures.Status != "valid" {
		return nil
	}
	var out []ChecklistLine
	for _, r := range trust.Policy.Reasons {
		if strings.EqualFold(r.Rule, "require-provenance") {
			continue
		}
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

func signatureLine(status string, policyReasons []apiclient.PolicyReason, signer *apiclient.TrustSigner) ChecklistLine {
	ruleID := policyRuleForSignatures(policyReasons)
	signedMsg := "✓ Signed by GitHub Actions"
	if signer != nil && signer.Repository != "" {
		signedMsg = fmt.Sprintf("✓ Signed by %s", signer.Repository)
		if signer.Workflow != "" {
			signedMsg += " (" + signer.Workflow + ")"
		}
		if signer.Ref != "" {
			signedMsg += " ref " + signer.Ref
		}
	}
	switch status {
	case "valid":
		return ChecklistLine{
			Text:          signedMsg,
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
