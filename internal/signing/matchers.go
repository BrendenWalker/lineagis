package signing

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

const githubActionsIssuer = `https://token\.actions\.githubusercontent\.com`

// defaultGitHubCertIdentity matches GitHub Actions workflow signing certificates.
const defaultGitHubCertIdentity = `https://github\.com/[^/]+/[^/]+/\.github/workflows/[^@]+@.*`

// PublisherMatcher describes one trusted Fulcio certificate identity (FR-SIGN-006).
type PublisherMatcher struct {
	Repository string
	Workflow   string
	Ref        string
	Issuer     string
}

// PermissiveKeylessIdentity reports whether dev-only permissive matchers are enabled.
func PermissiveKeylessIdentity() bool {
	v := strings.TrimSpace(os.Getenv("VERITY_PERMISSIVE_KEYLESS_IDENTITY"))
	return v == "1" || strings.EqualFold(v, "true")
}

// KeylessVerifyOptions builds cosign cert matchers from a namespace policy document.
func KeylessVerifyOptions(policyDocument json.RawMessage) VerifyOptions {
	if PermissiveKeylessIdentity() {
		return VerifyOptions{
			CertOidcIssuer: githubActionsIssuer,
			CertIdentity:   ".*",
		}
	}
	opts := VerifyOptions{
		CertOidcIssuer: githubActionsIssuer,
		CertIdentity:   defaultGitHubCertIdentity,
	}
	if id := certIdentityFromPolicy(policyDocument); id != "" {
		opts.CertIdentity = id
	}
	return opts
}

func certIdentityFromPolicy(document json.RawMessage) string {
	publishers := trustedPublishersFromPolicy(document)
	if len(publishers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(publishers))
	for _, p := range publishers {
		if re := certIdentityRegexp(p); re != "" {
			parts = append(parts, re)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, "|") + ")"
}

func trustedPublishersFromPolicy(document json.RawMessage) []PublisherMatcher {
	var doc struct {
		Rules []struct {
			ID     string          `json:"id"`
			Type   string          `json:"type"`
			Config json.RawMessage `json:"config"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(document, &doc); err != nil {
		return nil
	}
	var out []PublisherMatcher
	for _, rule := range doc.Rules {
		if !ruleIsTrustedPublishers(rule.ID, rule.Type) {
			continue
		}
		var cfg struct {
			Publishers []PublisherMatcher `json:"publishers"`
		}
		if len(rule.Config) > 0 {
			_ = json.Unmarshal(rule.Config, &cfg)
		}
		out = append(out, cfg.Publishers...)
	}
	return out
}

func ruleIsTrustedPublishers(id, typ string) bool {
	for _, v := range []string{id, typ} {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "trusted-publishers" || v == "trusted-publisher" {
			return true
		}
	}
	return false
}

// certIdentityRegexp returns a Fulcio SAN regexp for a trusted publisher entry.
func certIdentityRegexp(p PublisherMatcher) string {
	repo := strings.TrimSpace(p.Repository)
	wf := strings.TrimSpace(p.Workflow)
	if repo == "" && wf == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`https://github\.com/`)
	if repo != "" {
		b.WriteString(regexp.QuoteMeta(repo))
	} else {
		b.WriteString(`[^/]+/[^/]+`)
	}
	b.WriteString(`/\.github/workflows/`)
	if wf != "" {
		b.WriteString(regexp.QuoteMeta(wf))
	} else {
		b.WriteString(`[^@]+`)
	}
	ref := strings.TrimSpace(p.Ref)
	switch {
	case ref == "":
		b.WriteString(`@.*`)
	case strings.HasSuffix(ref, "*"):
		prefix := regexp.QuoteMeta(strings.TrimSuffix(ref, "*"))
		b.WriteString(`@` + prefix + `.*`)
	default:
		b.WriteString(`@` + regexp.QuoteMeta(ref))
	}
	return b.String()
}
