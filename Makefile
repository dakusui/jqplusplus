# Makefile for cmd (standard Go project layout)

BINARY_NAME=jq++
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=cmd/jqplusplus

# GOROOT can be set via environment variable or uncomment the line below
# GOROOT ?= /usr/local/go

# If GOROOT is set, use it for go commands
ifdef GOROOT
	GO := $(GOROOT)/bin/go
else
	GO := go
endif

.PHONY: all build run clean fmt

all: build

build:
	@mkdir -p bin
	$(GO) build -ldflags\
	 "-X main.version=$(shell tools/bin/version) -X main.revision=$(shell tools/bin/revision)"\
	 -o $(BINARY_PATH) ./$(CMD_PATH)

run:
	$(GO) run ./$(CMD_PATH)

doc:
	gendoc

clean:
	rm -rf bin/*
	rm -rf docs/*.html

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./... 
