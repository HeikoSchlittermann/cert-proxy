package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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

func TestFetchCNs_RejectsInvalidDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "valid.example.com")
		_, _ = fmt.Fprintln(w, "../../etc/passwd")
	}))
	defer srv.Close()

	origConnect := opt.Connect

	t.Cleanup(func() { opt.Connect = origConnect })

	opt.Connect = srv.URL

	_, err := fetchCNs(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid domain")
}

func TestCheckCertbase(t *testing.T) {
	base := t.TempDir()

	file := filepath.Join(base, "afile")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	tests := []struct {
		name string
		dir  string
		must string
	}{
		{"existing directory", base, ""},
		{"missing", filepath.Join(base, "absent"), "does not exist"},
		{"missing, nested", filepath.Join(base, "a", "b"), "does not exist"},
		{"a file", file, "not a directory"},
	}

	link := filepath.Join(base, "link")
	if err := os.Symlink(base, link); err == nil {
		tests = append(tests, struct {
			name string
			dir  string
			must string
		}{"symlink to a directory", link, ""})
	} else if runtime.GOOS == "windows" {
		t.Logf("symlink case unavailable without Windows symlink privileges: %v", err)
	} else {
		require.NoError(t, err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCertbase(tc.dir)
			if tc.must == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.must)
			assert.Contains(t, err.Error(), tc.dir, "the message must name the path")
		})
	}
}

// TestWithScheme covers the -connect shorthands. url.Parse reads "host:4433"
// as scheme "host" with opaque "4433", which used to make the client request
// "host:4433/v1/..." instead of talking to that host.
func TestWithScheme(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://host:4433", "https://host:4433"},
		{"http://host", "http://host"},
		{"host", "https://host"},
		{"host:4433", "https://host:4433"},
		{"//host:4433", "https://host:4433"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, withScheme(tc.in))
		})
	}
}

// TestConnectShorthandParses is the end-to-end form: what withScheme feeds to
// url.Parse must come back out as a usable base URL.
func TestConnectShorthandParses(t *testing.T) {
	for _, in := range []string{"host:4433", "host", "//host:4433", "https://host:4433"} {
		u, err := url.Parse(withScheme(in))
		require.NoError(t, err)
		assert.Equal(t, "host", u.Hostname(), in)
		assert.Empty(t, u.Opaque, "%s must not parse as an opaque URL", in)
		assert.NoError(t, checkConnectURL(u))
	}
}

func TestCheckConnectURL(t *testing.T) {
	tests := []struct {
		name, raw, want string
	}{
		{"https", "https://host:4433/base", ""},
		{"http", "http://host", ""},
		{"query", "https://host?token=x", "must not contain a query"},
		{"empty query", "https://host?", "must not contain a query"},
		{"fragment", "https://host#part", "must not contain a fragment"},
		{"missing host", "https:", "must name a server"},
		{"user information", "https://user:secret@host", "must not contain user information"},
		{"unsupported scheme", "file:///tmp/socket", "scheme must be http or https"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.raw)
			require.NoError(t, err)

			err = checkConnectURL(u)
			if tc.want == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
