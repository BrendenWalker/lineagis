package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1"
)

// PushBlob uploads raw blob bytes using the OCI Distribution blob upload API.
// Identical content yields the same digest; an existing blob is not re-uploaded.
func (c *Client) PushBlob(ctx context.Context, repo string, data []byte) (v1.Hash, error) {
	if len(data) > MaxBlobSize {
		return v1.Hash{}, fmt.Errorf("%w (%d bytes, limit %d)", ErrBlobTooLarge, len(data), MaxBlobSize)
	}

	h, _, err := v1.SHA256(bytes.NewReader(data))
	if err != nil {
		return v1.Hash{}, fmt.Errorf("registry: compute digest: %w", err)
	}

	exists, err := c.BlobExists(ctx, repo, h)
	if err != nil {
		return v1.Hash{}, err
	}
	if exists {
		return h, nil
	}

	if err := c.uploadBlob(ctx, repo, data, h); err != nil {
		return v1.Hash{}, err
	}
	return h, nil
}

func (c *Client) uploadBlob(ctx context.Context, repo string, data []byte, h v1.Hash) error {
	startURL, err := c.uploadsURL(repo)
	if err != nil {
		return err
	}

	startReq, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, nil)
	if err != nil {
		return fmt.Errorf("registry: create upload session: %w", err)
	}
	startReq.ContentLength = 0

	startResp, err := c.httpClient.Do(startReq)
	if err != nil {
		return fmt.Errorf("registry: start blob upload: %w", err)
	}
	defer func() { _ = startResp.Body.Close() }()

	if startResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("registry: start blob upload: %s", readStatus(startResp))
	}

	location := startResp.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("registry: start blob upload: missing Location header")
	}

	uploadURL, err := c.resolveReference(location)
	if err != nil {
		return err
	}
	uploadURL, err = appendDigestParam(uploadURL, h.String())
	if err != nil {
		return err
	}

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("registry: create upload request: %w", err)
	}
	putReq.Header.Set("Content-Type", "application/octet-stream")
	putReq.ContentLength = int64(len(data))

	putResp, err := c.httpClient.Do(putReq)
	if err != nil {
		return fmt.Errorf("registry: upload blob: %w", err)
	}
	defer func() { _ = putResp.Body.Close() }()

	if putResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registry: upload blob: %s", readStatus(putResp))
	}
	return nil
}

// PullBlob downloads blob bytes for the given digest and verifies content matches.
func (c *Client) PullBlob(ctx context.Context, repo string, h v1.Hash) ([]byte, error) {
	blobURL, err := c.blobURL(repo, h.String())
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("registry: create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry: download blob: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry: download blob: %s", readStatus(resp))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("registry: read blob: %w", err)
	}
	if len(data) > MaxBlobSize {
		return nil, fmt.Errorf("%w (%d bytes, limit %d)", ErrBlobTooLarge, len(data), MaxBlobSize)
	}

	got, _, err := v1.SHA256(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("registry: verify digest: %w", err)
	}
	if got != h {
		return nil, fmt.Errorf("registry: blob digest mismatch: got %s, want %s", got, h)
	}

	return data, nil
}

// BlobExists reports whether a blob with the given digest is present in the repository.
func (c *Client) BlobExists(ctx context.Context, repo string, h v1.Hash) (bool, error) {
	blobURL, err := c.blobURL(repo, h.String())
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, blobURL, nil)
	if err != nil {
		return false, fmt.Errorf("registry: create head request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("registry: head blob: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registry: head blob: %s", readStatus(resp))
	}
}

func readStatus(resp *http.Response) string {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if len(body) == 0 {
		return resp.Status
	}
	return fmt.Sprintf("%s: %s", resp.Status, bytes.TrimSpace(body))
}
