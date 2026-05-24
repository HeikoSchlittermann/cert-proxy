package main

import (
	"net/http"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
)

func versionLine() string {
	return program.Name + " " + program.Version + " " + program.Path
}

func versionCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add(`x-version`, program.Version)
}
