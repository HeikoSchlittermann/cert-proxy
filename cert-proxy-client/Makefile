.PHONY: all clean
v = $(shell git describe --always --dirty=+)
all: ; go build -ldflags '-X main.Version=$v'
clean: ; go clean
