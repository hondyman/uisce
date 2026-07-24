# Makefile for building the temporal-ops helper
BINARY := temporal-ops

.PHONY: all build install clean

all: build

build:
	@echo "Building $(BINARY) (GOWORK=off)..."
	GOWORK=off go build -v -o $(BINARY)

install: build
	@echo "Installing $(BINARY) to $(GOBIN) (if set)..."
	@if [ -n "$(GOBIN)" ]; then mv $(BINARY) $(GOBIN)/$(BINARY); else mv $(BINARY) $(GOPATH)/bin/$(BINARY) || true; fi

clean:
	@rm -f $(BINARY)
