SERVER = server
CLIENT = client

SUBDIRS = cmd/cert-proxy-${SERVER} cmd/cert-proxy-${CLIENT} CA

.PHONY: all test clean distclean install install-${SERVER} install-${CLIENT} install-ca

all build clean distclean: ; @for d in ${SUBDIRS}; do make -C $$d $@; done

test:
	go test ./...

update:
	GOWORK=off go get -u ./...
	go mod tidy

install:	     install-client install-server
install-client:      ; make -C cmd/cert-proxy-${CLIENT} install
install-server:      ; make -C cmd/cert-proxy-${SERVER} install
install-ca:	     ; make -C CA install
