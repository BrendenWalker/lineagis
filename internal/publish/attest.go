package publish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/apiclient"
	"github.com/BrendenWalker/lineagis/internal/provenance"
)

// SkipProvenanceFromEnv reports whether provenance generation is disabled.
func SkipProvenanceFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LINEAGIS_SKIP_PROVENANCE")))
	return v == "1" || v == "true" || v == "yes"
}

// attachProvenance generates, signs, and registers SLSA provenance (FR-PROV-005).
func attachProvenance(ctx context.Context, api *apiclient.Client, signer ManifestSigner, namespace, artifact, digest string) error {
	bctx := provenance.LoadBuildContext(digest)
	stmt, err := provenance.BuildSLSAStatement(bctx)
	if err != nil {
		return err
	}
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		return err
	}
	bundle, _, _, err := signer.SignManifest(ctx, statementJSON)
	if err != nil {
		return fmt.Errorf("sign provenance: %w", err)
	}
	return api.AttachAttestation(ctx, namespace, artifact, digest, provenance.PredicateSLSAProvenanceV1, statementJSON, bundle)
}

// attachSBOM reads an SBOM file, signs it, and registers the attestation (FR-PROV-007).
func attachSBOM(ctx context.Context, api *apiclient.Client, signer ManifestSigner, namespace, artifact, digest, sbomPath string) error {
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return fmt.Errorf("read sbom: %w", err)
	}
	predicateType, err := provenance.SBOMPredicateType(data)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	sbomDigest := map[string]string{"sha256": hex.EncodeToString(sum[:])}
	stmt := provenance.BuildSBOMStatement(digest, predicateType, sbomPath, sbomDigest)
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		return err
	}
	bundle, _, _, err := signer.SignManifest(ctx, statementJSON)
	if err != nil {
		return fmt.Errorf("sign sbom attestation: %w", err)
	}
	return api.AttachAttestation(ctx, namespace, artifact, digest, predicateType, statementJSON, bundle)
}
