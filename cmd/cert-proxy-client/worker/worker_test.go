package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/cert"
	"go.schlittermann.de/heiko/cert-proxy/internal/list"
)

func TestNewPool(t *testing.T) {
	pool := NewPool(context.Background(), 2)
	assert.NotNil(t, pool)
}

func TestPool_EnqueueAndWait_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("DATA"))
	}))
	defer srv.Close()

	origSymlink := cert.UseSymlink
	origForce := cert.Force
	cert.UseSymlink = false
	cert.Force = true

	t.Cleanup(func() {
		cert.UseSymlink = origSymlink
		cert.Force = origForce
	})

	basedir := t.TempDir()
	cns := list.UniqStrings{}
	cns.Add("a.example.com", "b.example.com")

	pool := NewPool(context.Background(), 2)
	require.NoError(t, pool.EnqueueTasks(cns, srv.URL, basedir, "", cert.FormatPEM, "", ""))

	err := pool.Wait()
	require.NoError(t, err)
}

func TestPool_EnqueueAndWait_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer srv.Close()

	origForce := cert.Force
	cert.Force = true

	t.Cleanup(func() {
		cert.Force = origForce
	})

	basedir := t.TempDir()
	cns := list.UniqStrings{}
	cns.Add("fail.example.com")

	pool := NewPool(context.Background(), 1)
	require.NoError(t, pool.EnqueueTasks(cns, srv.URL, basedir, "", cert.FormatPEM, "", ""))

	err := pool.Wait()
	assert.Error(t, err)
}

func TestPool_EnqueueTasks_InvalidURL(t *testing.T) {
	cns := list.UniqStrings{}
	cns.Add("example.com")

	pool := NewPool(context.Background(), 1)

	// A control character in the proxy URL makes http.NewRequestWithContext
	// fail inside cert.NewReq. The previous implementation panicked from a
	// worker goroutine; the new contract is to return the error.
	err := pool.EnqueueTasks(cns, "http://proxy:4433/\nbad", t.TempDir(), "", cert.FormatPEM, "", "")
	require.Error(t, err)

	// Wait must not deadlock — EnqueueTasks closed the queue on error.
	require.NoError(t, pool.Wait())
}

func TestPlural(t *testing.T) {
	assert.Equal(t, "s", plural(0))
	assert.Equal(t, "", plural(1))
	assert.Equal(t, "s", plural(2))
}
