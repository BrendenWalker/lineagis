package auth

import (
	"encoding/json"
	"fmt"
	"strings"
)

// githubNamespaceConfig is optional namespace.config for GitHub OIDC rules.
type githubNamespaceConfig struct {
	Repository  string   `json:"repository"`
	AllowedRefs []string `json:"allowed_refs"`
}

// ExpectedRepository returns the GitHub repository (owner/repo) required for namespace ns.
func ExpectedRepository(ns string, config json.RawMessage) (string, bool) {
	var cfg githubNamespaceConfig
	if len(config) > 0 && string(config) != "{}" {
		_ = json.Unmarshal(config, &cfg)
	}
	if cfg.Repository != "" {
		return cfg.Repository, true
	}
	const prefix = "gh/"
	if strings.HasPrefix(ns, prefix) {
		repo := strings.TrimPrefix(ns, prefix)
		if repo != "" && !strings.HasPrefix(repo, "/") {
			return repo, true
		}
	}
	return "", false
}

// AuthorizeNamespace checks GitHub OIDC claims against namespace rules (FR-API-003).
// Dev actors skip claim checks. Non-GitHub namespaces skip until publisher rules exist.
func AuthorizeNamespace(actor Actor, ns string, config json.RawMessage) error {
	if actor.Dev || actor.GitHub == nil {
		return nil
	}
	expectedRepo, enforceRepo := ExpectedRepository(ns, config)
	if !enforceRepo {
		return nil
	}
	if actor.GitHub.Repository != expectedRepo {
		return fmt.Errorf("repository claim %q does not match namespace %q", actor.GitHub.Repository, expectedRepo)
	}

	var cfg githubNamespaceConfig
	if len(config) > 0 {
		_ = json.Unmarshal(config, &cfg)
	}
	if len(cfg.AllowedRefs) == 0 {
		return nil
	}
	for _, allowed := range cfg.AllowedRefs {
		if actor.GitHub.Ref == allowed {
			return nil
		}
	}
	return fmt.Errorf("ref claim %q is not allowed for namespace %q", actor.GitHub.Ref, ns)
}
