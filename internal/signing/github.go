package signing

import (
	"crypto/x509"
	"strings"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// GitHubPublisher holds workflow identity from a Fulcio certificate (FR-SIGN-008, FR-POL-006).
type GitHubPublisher struct {
	Repository string
	Workflow   string
	Ref        string
}

// GitHubPublisherFromCertificate extracts GitHub Actions workflow identity when present.
func GitHubPublisherFromCertificate(cert *x509.Certificate) (GitHubPublisher, bool) {
	if cert == nil {
		return GitHubPublisher{}, false
	}
	ext := cosign.CertExtensions{Cert: cert}
	repo := strings.TrimSpace(ext.GetCertExtensionGithubWorkflowRepository())
	if repo == "" {
		return GitHubPublisher{}, false
	}
	return GitHubPublisher{
		Repository: repo,
		Workflow:   strings.TrimSpace(ext.GetCertExtensionGithubWorkflowName()),
		Ref:        strings.TrimSpace(ext.GetCertExtensionGithubWorkflowRef()),
	}, true
}

// GitHubPublisherFromBundle extracts GitHub workflow identity from a Sigstore bundle cert.
func GitHubPublisherFromBundle(bundleJSON []byte) (GitHubPublisher, bool) {
	pem := LegacyBundleCertPEM(bundleJSON)
	if len(pem) == 0 {
		return GitHubPublisher{}, false
	}
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(pem)
	if err != nil || len(certs) == 0 {
		return GitHubPublisher{}, false
	}
	return GitHubPublisherFromCertificate(certs[0])
}
