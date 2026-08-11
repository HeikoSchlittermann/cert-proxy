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

MANPAGES = ${BUILDDIR}/man/cert-proxy-client.8 ${BUILDDIR}/man/cert-proxy-server.8

.PHONY: all build test update install install-client install-server install-ca man clean

all: build

# The version reported by both binaries comes from the Go build info
# (runtime/debug), so no -ldflags -X is needed here.
build:
	go build -o ${BUILDDIR}/ ./cmd/...

test:
	go test ./...

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

man: ${MANPAGES}

${BUILDDIR}/man/cert-proxy-client.8: cmd/cert-proxy-client/cert-proxy-client.pod
	@install -d ${@D}
	pod2man --release "" --center schlittermann --section 8 $< $@

${BUILDDIR}/man/cert-proxy-server.8: cmd/cert-proxy-server/cert-proxy-server.pod
	@install -d ${@D}
	pod2man --release "" --center schlittermann --section 8 $< $@

clean:
	go clean ./...
	rm -rf ${BUILDDIR}
