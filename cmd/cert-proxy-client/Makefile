.PHONY: all build install clean
PACKAGE = cert-proxy-client

include ../.makefile.common

cert-proxy-client.8:	cert-proxy-client.pod
	pod2man --release "" --center schlittermann --section 8 $? $@
