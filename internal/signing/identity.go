package signing

import (
	"crypto/x509"
	"strings"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// Identity holds OIDC issuer and subject extracted from a Fulcio certificate (FR-SIGN-008).
type Identity struct {
	Issuer  string
	Subject string
}

// IdentityFromCertificatePEM parses issuer and subject from a Fulcio signing certificate.
func IdentityFromCertificatePEM(pemBytes []byte) (Identity, error) {
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(pemBytes)
	if err != nil {
		return Identity{}, err
	}
	if len(certs) == 0 {
		return Identity{}, nil
	}
	return IdentityFromCertificate(certs[0]), nil
}

// IdentityFromCertificate extracts OIDC issuer and signer subject from a Fulcio leaf cert.
func IdentityFromCertificate(cert *x509.Certificate) Identity {
	ext := cosign.CertExtensions{Cert: cert}
	issuer := ext.GetIssuer()
	subject := strings.Join(cryptoutils.GetSubjectAlternateNames(cert), " ")
	if subject == "" {
		subject = cert.Subject.String()
	}
	return Identity{Issuer: issuer, Subject: subject}
}
