BINARY   := tubectl
CMD_PATH := .
INSTALL_DIR := $(shell go env GOPATH)/bin

build: test
	mkdir -p bin
	go build -o bin/$(BINARY) $(CMD_PATH)

install: test
	go install $(CMD_PATH)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

test:
	go test ./...

clean:
	rm -rf bin/

.PHONY: build install test clean
