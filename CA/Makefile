DESTDIR ?=
.PHONY: all clean build install

all clean build:
install:
	@for dir in bin lib; do \
		install -d ${DESTDIR}/etc/cert-proxy/ca/$$dir; \
		install $$dir/* ${DESTDIR}/etc/cert-proxy/ca/$$dir ;\
	done
