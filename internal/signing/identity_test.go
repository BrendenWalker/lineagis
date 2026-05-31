package signing_test

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"net/url"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/signing"
)

func TestIdentityFromCertificate_githubOIDC(t *testing.T) {
	t.Parallel()
	workflowURI, err := url.Parse("https://github.com/acme/widget/.github/workflows/release.yml@refs/heads/main")
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{
		Extensions: []pkix.Extension{
			{Id: fulcioOID(1), Value: []byte("https://token.actions.githubusercontent.com")},
			{Id: fulcioOID(5), Value: []byte("acme/widget")},
			{Id: fulcioOID(6), Value: []byte("refs/heads/main")},
		},
		URIs: []*url.URL{workflowURI},
	}
	id := signing.IdentityFromCertificate(cert)
	if id.Issuer != "https://token.actions.githubusercontent.com" {
		t.Fatalf("issuer = %q", id.Issuer)
	}
	if id.Subject == "" {
		t.Fatal("expected non-empty subject from SAN")
	}
}

// fulcioOID returns 1.3.6.1.4.1.57264.1.<n> (Fulcio GitHub extension arc).
func fulcioOID(n int) asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, n}
}
