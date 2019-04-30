.PHONY: all clean
v = $(shell git describe --always --dirty=+)
d = $(shell date)
all: ; go build -ldflags '-X main.Version=$v'
clean: ; go clean
