// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

// errInvalidCompat is returned for a pkcs12-compat value other than
// "modern", "legacy" or empty.
var errInvalidCompat = errors.New("invalid pkcs12-compat")

// pkcs12Encoder maps the pkcs12-compat request parameter to an encoder.
// Modern is the default; the weak legacy encoding needs an explicit request.
func pkcs12Encoder(compat string) (*pkcs12.Encoder, error) {
	switch compat {
	case "modern", "":
		return pkcs12.Modern2023, nil
	case "legacy":
		return pkcs12.LegacyDES, nil
	default:
		return nil, fmt.Errorf("%w %q", errInvalidCompat, compat)
	}
}

func createPKCS12(certbase, domain, pass, compat string) (*bytes.Reader, time.Time, error) {
	encoder, err := pkcs12Encoder(compat)
	if err != nil {
		return nil, time.Time{}, err
	}

	var (
		certPath  = filepath.Join(certbase, domain, "cert.pem")
		keyPath   = filepath.Join(certbase, domain, "privkey.pem")
		chainPath = filepath.Join(certbase, domain, "chain.pem")
		mtime     time.Time
	)

	fi, err := os.Stat(certPath)
	if err != nil {
		return nil, mtime, err
	}

	mtime = fi.ModTime()

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, mtime, err
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, mtime, err
	}

	chainPEM, err := os.ReadFile(chainPath)
	if err != nil {
		return nil, mtime, err
	}

	privateKey, err := parsePrivateKey(keyPEM)
	if err != nil {
		return nil, mtime, fmt.Errorf("parsing private key: %w", err)
	}

	cert, err := parseCertificate(certPEM)
	if err != nil {
		return nil, mtime, fmt.Errorf("parsing certificate: %w", err)
	}

	chainCerts, err := parseCertificates(chainPEM)
	if err != nil {
		return nil, mtime, fmt.Errorf("parsing chain: %w", err)
	}

	pfxData, err := encoder.Encode(privateKey, cert, chainCerts, pass)
	if err != nil {
		return nil, mtime, fmt.Errorf("encoding PKCS12: %w", err)
	}

	return bytes.NewReader(pfxData), mtime, nil
}

func parsePrivateKey(pemData []byte) (any, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("unsupported private key type")
}

func parseCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	return x509.ParseCertificate(block.Bytes)
}

func parseCertificates(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		certs = append(certs, cert)
		pemData = rest
	}

	return certs, nil
}
