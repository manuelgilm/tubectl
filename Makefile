BINARY=tubectl

build:
	go build -o $(BINARY) .

install:
	go install .

clean:
	rm -f $(BINARY)

.PHONY: build install clean

BINARY   := tubectl
CMD_PATH := ./cmd/tubectl
INSTALL_DIR := $(shell go env GOPATH)/bin

.PHONY: build install test clean

