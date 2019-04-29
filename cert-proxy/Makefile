DESTDIR = /
.PHONY: all clean
all: ; go build
clean: ; go clean

install:	all
	install -m 0755 -d ${DESTDIR}/etc/cert-proxy
