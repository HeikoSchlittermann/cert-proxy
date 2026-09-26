package main

import (
	"net/http"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
)

func versionLine() string {
	return program.Name + " " + program.Version + " " + program.Path
}

// versionCheck announces the server version, but only to authenticated
// clients: either authn already ran, or the client presented a certificate
// that passed verification (tls.VerifyClientCertIfGiven). Anonymous clients
// get no version fingerprint.
func versionCheck(ctx context, w http.ResponseWriter, req *http.Request) {
	if ctx[REMOTE] == "" && (req.TLS == nil || len(req.TLS.PeerCertificates) == 0) {
		return
	}

	w.Header().Add(`x-version`, program.Version)
}
