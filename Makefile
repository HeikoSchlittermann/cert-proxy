SUBDIRS = cert-proxy cert-proxy-client
.PHONY: all clean

all clean:	; @for d in ${SUBDIRS}; do make -C $$d $@; done
