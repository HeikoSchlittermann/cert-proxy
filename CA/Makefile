DESTDIR ?=
.PHONY: all clean distclean build install

all clean build distclean:
install:
	@for dir in bin lib; do \
		install -d ${DESTDIR}/etc/cert-proxy/ca/$$dir; \
		install $$dir/* ${DESTDIR}/etc/cert-proxy/ca/$$dir ;\
	done
