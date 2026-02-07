# Makefile for cmd (standard Go project layout)

BINARY_NAME=jqplusplus
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


.PHONY: all
all: build

.PHONY: build
build:
	@mkdir -p bin
	$(GO) build -ldflags\
	 "-X main.version=$(shell tools/bin/version) -X main.revision=$(shell tools/bin/revision)"\
	 -o $(BINARY_PATH) ./$(CMD_PATH)
	$(BINARY_PATH)

.PHONY: run
run:
	$(GO) run ./$(CMD_PATH)

.PHONY: doc
doc:
	gendoc

.PHONY: pubdoc
pubdoc:
	tools/bin/pubdoc

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf docs/*.html

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: test
test:
	$(GO) test ./... 
