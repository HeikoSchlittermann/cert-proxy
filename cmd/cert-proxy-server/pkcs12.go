package main

import (
	"bytes"
	. "git.schlittermann.de/user/heiko/cert-proxy.git/shared"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func createPKCS12(certbase, domain, pass string) (*bytes.Reader, time.Time, error) {

	var cert = filepath.Join(certbase, domain, `cert.pem`)
	var key = filepath.Join(certbase, domain, `privkey.pem`)
	var chain = filepath.Join(certbase, domain, `chain.pem`)
	var mtime time.Time

	if fi, err := os.Stat(cert); err != nil {
		return nil, mtime, err
	} else {
		mtime = fi.ModTime()
	}

	var cmd = exec.Command(`openssl`, `pkcs12`,
		`-export`,
		`-passout`, `pass:`+pass,
		`-inkey`, key,
		`-in`, cert,
		`-certfile`, chain)
	Verbose("Starting %s", cmd.Path)
	pkcs12, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		log.Printf("%s %v: %s", cmd.Path, cmd.Args, err.Stderr)
	}
	return bytes.NewReader(pkcs12), mtime, err
}
