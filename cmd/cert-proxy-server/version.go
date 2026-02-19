// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
)

func versionCheck(w http.ResponseWriter, _ *http.Request) {
	/* FIXME: Do not compare the versions, but establish
	if program.Version != req.Header.Get(`x-version`) {
		log.Printf("WARNING: Version mismatch: server:%s client:%s",
			program.Version, req.Header.Get(`x-version`))
	}
	*/
	w.Header().Add(`x-version`, program.Version)
}
