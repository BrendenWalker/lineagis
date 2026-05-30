package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/BrendenWalker/verity/internal/metadata"
)

func TestDeliverWebhook_hmac(t *testing.T) {
	var (
		mu      sync.Mutex
		gotBody []byte
		gotSig  string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotBody = body
		gotSig = r.Header.Get("X-Verity-Signature")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	secret := "shh"
	ep := metadata.WebhookEndpoint{
		Name:   "ci",
		URL:    srv.URL,
		Secret: &secret,
	}
	body := []byte(`{"event_type":"tag.set"}`)
	if err := deliverWebhook(&http.Client{Timeout: 2 * time.Second}, ep, body); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if string(gotBody) != string(body) {
		t.Fatalf("body %q", gotBody)
	}
	if gotSig == "" {
		t.Fatal("expected X-Verity-Signature")
	}
}
