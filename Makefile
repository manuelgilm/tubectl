BINARY   := tubectl
CMD_PATH := ./cmd/tubectl
INSTALL_DIR := $(shell go env GOPATH)/bin

.PHONY: build install test clean

## build: compile the binary into ./bin/tubectl
build:
	mkdir -p bin
	go build -o bin/$(BINARY) $(CMD_PATH)

## install: build and install the binary to $GOPATH/bin
install:
	go install $(CMD_PATH)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

## test: run all tests
test:
	go test ./...

## clean: remove the local build output
clean:
	rm -rf bin/
