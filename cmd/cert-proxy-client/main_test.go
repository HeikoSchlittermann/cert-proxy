package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests mutate the global opt.Connect and cannot run in parallel.

func TestFetchCNs_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate what the server does for a client with no domains assigned:
		// fmt.Fprintln(w, strings.Join(nil, "\n")) produces just "\n"
		fmt.Fprintln(w, "")
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	domains, err := fetchCNs()
	require.NoError(t, err)
	assert.Empty(t, domains, "empty server response should yield no domains")
}

func TestFetchCNs_WithDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "example.com\nsub.example.com")
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	domains, err := fetchCNs()
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

	_, err := fetchCNs()
	assert.Error(t, err)
}
