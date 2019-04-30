SERVER = server
CLIENT = client

SUBDIRS = cert-proxy-${SERVER} cert-proxy-${CLIENT} CA

.PHONY: all clean install install-${SERVER} install-${CLIENT} install-ca

all clean:	; @for d in ${SUBDIRS}; do make -C $$d $@; done

install:	install-client install-server

install-client: ; make -C cert-proxy-${CLIENT} install
install-server: ; make -C cert-proxy-${SERVER} install
install-ca:	; make -C CA install
