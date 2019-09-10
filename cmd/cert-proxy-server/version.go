package main

import (
	"cert-proxy/internal/program"
	"net/http"
)

func versionCheck(w http.ResponseWriter, req *http.Request) {
	/* FIXME: Do not compare the versions, but establish
	if program.Version != req.Header.Get(`x-version`) {
		log.Printf("WARNING: Version mismatch: server:%s client:%s",
			program.Version, req.Header.Get(`x-version`))
	}
	*/
	w.Header().Add(`x-version`, program.Version)
}
