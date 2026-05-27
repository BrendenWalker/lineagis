package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	PredicateSLSAProvenanceV1 = "https://slsa.dev/provenance/v1"
	PredicateSPDX             = "https://spdx.dev/Document"
	PredicateCycloneDX        = "https://cyclonedx.org/bom"
)

// Fields are parsed provenance index values (FR-PROV-010).
type Fields struct {
	RepositoryURI string
	CommitSHA     string
	WorkflowName  string
	WorkflowRef   string
	RunID         string
}

// BuildContext collects publish-time provenance inputs (FR-PROV-002, FR-PROV-003).
type BuildContext struct {
	ManifestDigest string
	RepositoryURI  string
	CommitSHA      string
	WorkflowName   string
	WorkflowRef    string
	RunID          string
	BuilderID      string
}

// LoadBuildContext reads git and GitHub Actions environment when available.
func LoadBuildContext(manifestDigest string) BuildContext {
	ctx := BuildContext{
		ManifestDigest: strings.TrimSpace(manifestDigest),
		BuilderID:      "https://github.com/actions/runner",
	}
	if repo := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY")); repo != "" {
		ctx.RepositoryURI = "https://github.com/" + repo
	}
	ctx.CommitSHA = firstNonEmpty(
		os.Getenv("GITHUB_SHA"),
		gitOutput("rev-parse", "HEAD"),
	)
	ctx.WorkflowName = strings.TrimSpace(os.Getenv("GITHUB_WORKFLOW"))
	ctx.WorkflowRef = firstNonEmpty(
		os.Getenv("GITHUB_REF"),
		os.Getenv("GITHUB_REF_NAME"),
	)
	ctx.RunID = strings.TrimSpace(os.Getenv("GITHUB_RUN_ID"))
	if strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true") {
		ctx.BuilderID = "https://github.com/Attestations/GitHubActions/trusted"
	}
	return ctx
}

// Statement is an in-toto v1 statement (FR-PROV-001).
type Statement struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}

type Subject struct {
	Name   string          `json:"name"`
	Digest map[string]string `json:"digest"`
}

type Predicate struct {
	BuildDefinition BuildDefinition `json:"buildDefinition"`
	RunDetails      RunDetails      `json:"runDetails"`
}

type BuildDefinition struct {
	BuildType string          `json:"buildType"`
	ExternalParameters map[string]any `json:"externalParameters,omitempty"`
	ResolvedDependencies []ResolvedDependency `json:"resolvedDependencies,omitempty"`
}

type ResolvedDependency struct {
	URI     string            `json:"uri"`
	Digest  map[string]string `json:"digest"`
}

type RunDetails struct {
	Builder   BuilderInfo `json:"builder"`
	Metadata  RunMetadata `json:"metadata"`
}

type BuilderInfo struct {
	ID string `json:"id"`
}

type RunMetadata struct {
	InvocationID string `json:"invocationId,omitempty"`
	StartedOn    string `json:"startedOn,omitempty"`
	FinishedOn   string `json:"finishedOn,omitempty"`
}

// BuildSLSAStatement constructs a SLSA Provenance v1 in-toto statement (FR-PROV-001–004).
func BuildSLSAStatement(ctx BuildContext) (Statement, error) {
	if ctx.ManifestDigest == "" {
		return Statement{}, fmt.Errorf("provenance: manifest digest is required")
	}
	algo, hex, ok := strings.Cut(ctx.ManifestDigest, ":")
	if !ok || algo == "" || hex == "" {
		return Statement{}, fmt.Errorf("provenance: digest must be algorithm:hex")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	invocation := ctx.RunID
	if invocation == "" {
		invocation = "local"
	}

	var deps []ResolvedDependency
	if ctx.RepositoryURI != "" && ctx.CommitSHA != "" {
		deps = append(deps, ResolvedDependency{
			URI: ctx.RepositoryURI + "/.git",
			Digest: map[string]string{"gitCommit": ctx.CommitSHA},
		})
	}

	stmt := Statement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: PredicateSLSAProvenanceV1,
		Subject: []Subject{{
			Name:   ctx.RepositoryURI,
			Digest: map[string]string{algo: hex},
		}},
		Predicate: Predicate{
			BuildDefinition: BuildDefinition{
				BuildType: "https://verity.dev/slsa/v1/github-actions",
				ExternalParameters: map[string]any{
					"workflow": map[string]string{
						"name": ctx.WorkflowName,
						"ref":  ctx.WorkflowRef,
					},
				},
				ResolvedDependencies: deps,
			},
			RunDetails: RunDetails{
				Builder: BuilderInfo{ID: ctx.BuilderID},
				Metadata: RunMetadata{
					InvocationID: invocation,
					StartedOn:    now,
					FinishedOn:   now,
				},
			},
		},
	}
	return stmt, nil
}

// MarshalStatement returns canonical JSON for signing.
func MarshalStatement(stmt Statement) ([]byte, error) {
	return json.Marshal(stmt)
}

// ParseFields extracts index fields from a provenance statement (FR-PROV-010).
func ParseFields(stmt Statement) Fields {
	var out Fields
	out.RepositoryURI = strings.TrimSpace(stmt.Subject[0].Name)
	if len(stmt.Predicate.BuildDefinition.ResolvedDependencies) > 0 {
		dep := stmt.Predicate.BuildDefinition.ResolvedDependencies[0]
		if sha, ok := dep.Digest["gitCommit"]; ok {
			out.CommitSHA = sha
		}
		if out.RepositoryURI == "" {
			out.RepositoryURI = strings.TrimSuffix(dep.URI, "/.git")
		}
	}
	if wfRaw, ok := stmt.Predicate.BuildDefinition.ExternalParameters["workflow"]; ok {
		switch wf := wfRaw.(type) {
		case map[string]any:
			if v, ok := wf["name"].(string); ok {
				out.WorkflowName = v
			}
			if v, ok := wf["ref"].(string); ok {
				out.WorkflowRef = v
			}
		case map[string]string:
			out.WorkflowName = wf["name"]
			out.WorkflowRef = wf["ref"]
		}
	}
	out.RunID = stmt.Predicate.RunDetails.Metadata.InvocationID
	return out
}

// ParseStatement unmarshals an in-toto statement.
func ParseStatement(raw json.RawMessage) (Statement, error) {
	var stmt Statement
	if err := json.Unmarshal(raw, &stmt); err != nil {
		return Statement{}, fmt.Errorf("provenance: parse statement: %w", err)
	}
	if stmt.PredicateType == "" {
		return Statement{}, fmt.Errorf("provenance: missing predicateType")
	}
	return stmt, nil
}

// SBOMPredicateType detects SPDX or CycloneDX from file contents (FR-PROV-007).
func SBOMPredicateType(data []byte) (string, error) {
	trim := strings.TrimSpace(string(data))
	if trim == "" {
		return "", fmt.Errorf("provenance: sbom file is empty")
	}
	if strings.Contains(trim, `"spdxVersion"`) || strings.HasPrefix(trim, "SPDXVersion:") {
		return PredicateSPDX, nil
	}
	if strings.Contains(trim, `"bomFormat"`) && strings.Contains(trim, "CycloneDX") {
		return PredicateCycloneDX, nil
	}
	return "", fmt.Errorf("provenance: unsupported sbom format (want SPDX or CycloneDX)")
}

// BuildSBOMStatement wraps SBOM bytes in an in-toto statement bound to digest.
func BuildSBOMStatement(manifestDigest, predicateType string, sbomName string, sbomDigest map[string]string) Statement {
	algo, hex, _ := strings.Cut(manifestDigest, ":")
	subjectDigest := map[string]string{algo: hex}
	return Statement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: predicateType,
		Subject: []Subject{{
			Name:   manifestDigest,
			Digest: subjectDigest,
		}},
		Predicate: Predicate{
			BuildDefinition: BuildDefinition{
				BuildType: "https://verity.dev/sbom/v1",
				ExternalParameters: map[string]any{
					"sbom": map[string]any{
						"name":   sbomName,
						"digest": sbomDigest,
					},
				},
			},
			RunDetails: RunDetails{
				Builder: BuilderInfo{ID: "https://verity.dev/cli"},
			},
		},
	}
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
