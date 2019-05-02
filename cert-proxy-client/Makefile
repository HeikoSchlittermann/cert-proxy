.PHONY: all build install clean
v = $(shell git describe --always --dirty=+)
all:			build
build install clean:	; go $@ -ldflags '-X cert-proxy/internal/program.Version=$v'
