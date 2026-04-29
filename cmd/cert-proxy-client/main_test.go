package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests mutate the global opt.Connect and cannot run in parallel.

func TestFetchCNs_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate what the server does for a client with no domains assigned:
		// fmt.Fprintln(w, strings.Join(nil, "\n")) produces just "\n"
		_, _ = fmt.Fprintln(w, "")
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	domains, err := fetchCNs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, domains, "empty server response should yield no domains")
}

func TestFetchCNs_WithDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "example.com\nsub.example.com")
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	domains, err := fetchCNs(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "sub.example.com"}, domains)
}

func TestFetchCNs_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no config", http.StatusInternalServerError)
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	_, err := fetchCNs(context.Background())
	assert.Error(t, err)
}

func TestFetchCNs_ContextCancellation(t *testing.T) {
	block := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(block)

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() {
		_, err := fetchCNs(ctx)
		errCh <- err
	}()

	cancel()

	select {
	case err := <-errCh:
		assert.Error(t, err, "fetchCNs should return an error after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("fetchCNs did not return after context cancellation")
	}
}
