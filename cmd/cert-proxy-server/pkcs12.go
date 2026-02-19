// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

func createPKCS12(certbase, domain, pass string) (*bytes.Reader, time.Time, error) {
	var (
		cert  = filepath.Join(certbase, domain, `cert.pem`)
		key   = filepath.Join(certbase, domain, `privkey.pem`)
		chain = filepath.Join(certbase, domain, `chain.pem`)
		mtime time.Time
	)

	fi, err := os.Stat(cert)
	if err != nil {
		return nil, mtime, err
	}

	mtime = fi.ModTime()

	var cmd = exec.Command(`openssl`, `pkcs12`,
		`-export`,
		`-passout`, `pass:`+pass,
		`-inkey`, key,
		`-in`, cert,
		`-certfile`, chain)
	shared.Verbose("Starting %s", cmd.Path)

	pkcs12, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		log.Printf("%s %v: %s", cmd.Path, cmd.Args, err.Stderr)
	}

	return bytes.NewReader(pkcs12), mtime, err
}
