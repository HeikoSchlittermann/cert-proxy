# Local development helpers.
#
# Release artifacts (cross-built binaries, .deb) are produced by gogogo from
# .gogogo.conf, not from here. This file covers what gogogo does not do:
# building and installing into a local prefix, and installing the CA helper
# scripts on hosts that do not use the Debian package.

export GOWORK = off

prefix  ?= /usr/local
bindir  ?= ${DESTDIR}${prefix}/bin
cadir   ?= ${DESTDIR}/etc/cert-proxy/ca

BUILDDIR = build

ifeq (${GOOS},windows)
EXE     = .exe
bindir := ${bindir}/windows_$(shell go env GOARCH)
endif

# Listed explicitly rather than globbed: CA/lib/ also holds the admin's own
# vars.sh, and mkca creates the CA itself under CA_DIR. A glob would install
# whatever happens to be lying around.
CA_BIN  = CA/bin/mkssl-pem
CA_LIB  = CA/lib/mkca
CA_CONF = CA/lib/openssl.cnf CA/lib/vars.sh.example

.PHONY: all build test test-packaging test-packaging-image update install install-client install-server install-ca man clean

# Image used by the packaging cases that need systemd-sysusers and
# systemd-tmpfiles. Provisioning it is an operator concern, not the test's:
# the test skips those cases while the image is absent.
#
# Plain networks:
#   make test-packaging-image
# Behind an apt proxy on the host loopback, intercepting TLS:
#   make test-packaging-image APT_PROXY=http://localhost:3128/ \
#        APT_VERIFY_PEER=false PODMAN_BUILD_FLAGS=--network=host
PKGTEST_IMAGE    ?= localhost/cert-proxy-pkgtest
APT_PROXY        ?=
APT_VERIFY_PEER  ?= true
PODMAN_BUILD_FLAGS ?=

all: build

# The version reported by both binaries comes from the Go build info
# (runtime/debug), so no -ldflags -X is needed here.
build:
	go build -o ${BUILDDIR}/ ./cmd/...

test:
	go test ./...

# Opt-in: builds the .deb files with gogogo and installs them in a throwaway
# podman container. Needs podman and gogogo; the cases covering the
# sysusers/tmpfiles assets additionally need a one-off image build with
# Debian archive access, and skip themselves without it.
test-packaging:
	go test -tags packaging ./test/packaging/

test-packaging-image:
	podman build ${PODMAN_BUILD_FLAGS} \
	    --build-arg APT_PROXY='${APT_PROXY}' \
	    --build-arg APT_VERIFY_PEER='${APT_VERIFY_PEER}' \
	    -t ${PKGTEST_IMAGE} \
	    -f test/packaging/testdata/Containerfile test/packaging/testdata/

update:
	go get -t -u ./...
	go mod tidy

install: install-client install-server

install-client: build
	install -d ${bindir}
	install --strip ${BUILDDIR}/cert-proxy-client${EXE} ${bindir}/

install-server: build
	install -d ${bindir}
	install --strip ${BUILDDIR}/cert-proxy-server${EXE} ${bindir}/

install-ca:
	install -d ${cadir}/bin ${cadir}/lib
	install -m 0755 ${CA_BIN} ${cadir}/bin/
	install -m 0755 ${CA_LIB} ${cadir}/lib/
	install -m 0644 ${CA_CONF} ${cadir}/lib/

# The tracked man/*.gz pages are generated from man/*.md by man/gen.go
# through the pinned go-md2man tool; see AGENTS.md.
man:
	go generate ./...

clean:
	go clean ./...
	rm -rf ${BUILDDIR}
