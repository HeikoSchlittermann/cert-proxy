SERVER = server
CLIENT = client

SUBDIRS = cert-proxy-${SERVER} cert-proxy-${CLIENT}

.PHONY: all clean install install-${SERVER} install-${CLIENT}

all clean:	; @for d in ${SUBDIRS}; do make -C $$d $@; done

install:	install-client install-server

install-client: ; make -C cert-proxy-${CLIENT} install
install-server: ; make -C cert-proxy-${SERVER} install
