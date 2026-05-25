package auth

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Role is an MVP namespace-scoped permission (api.md).
type Role string

const (
	RoleOperator   Role = "operator"
	RoleMaintainer Role = "maintainer"
	RoleReader     Role = "reader"
)

type namespaceRolesConfig struct {
	Operators     []string `json:"operators"`
	AnonymousRead bool     `json:"anonymous_read"`
}

// IsOperator returns true when the actor may manage policy and namespace config (FR-API-004).
func IsOperator(actor Actor, config json.RawMessage) bool {
	if actor.Dev {
		return true
	}
	var cfg namespaceRolesConfig
	if len(config) > 0 {
		_ = json.Unmarshal(config, &cfg)
	}
	subject := strings.TrimSpace(actor.Subject)
	if subject == "" {
		return false
	}
	return slices.Contains(cfg.Operators, subject)
}

// AuthorizeRole checks namespace-scoped role membership.
func AuthorizeRole(actor Actor, ns string, config json.RawMessage, role Role) error {
	switch role {
	case RoleOperator:
		if !IsOperator(actor, config) {
			return fmt.Errorf("operator role required for namespace %q", ns)
		}
	case RoleMaintainer:
		if actor.Dev || IsOperator(actor, config) {
			return nil
		}
		if actor.GitHub != nil {
			return AuthorizeNamespace(actor, ns, config)
		}
		return fmt.Errorf("maintainer role required for namespace %q", ns)
	case RoleReader:
		if actor.Dev || IsOperator(actor, config) {
			return nil
		}
		if actor.Subject != "" {
			return nil
		}
		return fmt.Errorf("reader role required for namespace %q", ns)
	default:
		return fmt.Errorf("unknown role %q", role)
	}
	return nil
}

// AllowsAnonymousRead reports whether unauthenticated GET is permitted (FR-API-005).
func AllowsAnonymousRead(config json.RawMessage) bool {
	var cfg namespaceRolesConfig
	if len(config) > 0 {
		_ = json.Unmarshal(config, &cfg)
	}
	return cfg.AnonymousRead
}
