SUBDIRS = cert-proxy-server cert-proxy-client
.PHONY: all clean install install-client install-server

all clean:	; @for d in ${SUBDIRS}; do make -C $$d $@; done
