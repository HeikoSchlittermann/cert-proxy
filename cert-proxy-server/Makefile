.PHONY: all clean install
v = $(shell git describe --always --dirty=+)
all:		; go build -ldflags '-X main.Version=$v'
clean install:	; go $@
