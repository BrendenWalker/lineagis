package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubChecker struct {
	dbErr       error
	registryErr error
}

func (s stubChecker) PingDB(context.Context) error {
	return s.dbErr
}

func (s stubChecker) CheckRegistry(context.Context) error {
	return s.registryErr
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "ok" {
		t.Fatalf("GET /healthz body = %q, want %q", got, "ok")
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /healthz status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestReadyzOK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /readyz status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "ok" {
		t.Fatalf("GET /readyz body = %q, want %q", got, "ok")
	}
}

func TestReadyzDatabaseUnavailable(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{dbErr: errors.New("db down")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyzRegistryUnavailable(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{registryErr: errors.New("registry down")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyzMethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/readyz", nil)
	rec := httptest.NewRecorder()
	testMux(stubChecker{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /readyz status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestUnexpectedStatusError(t *testing.T) {
	t.Parallel()

	err := errUnexpectedStatus(http.StatusBadGateway)
	if err.Error() != "unexpected registry status: Bad Gateway" {
		t.Fatalf("Error() = %q", err.Error())
	}
}
