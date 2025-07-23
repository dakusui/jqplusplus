# Makefile for jqplusplus (standard Go project layout)

BINARY_NAME=jq++
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=jqplusplus/cmd/jqplusplus

.PHONY: all build run clean fmt

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY_PATH) ./$(CMD_PATH)

run:
	go run ./$(CMD_PATH)

doc:
	./tools/gendoc.sh

clean:
	rm -rf bin
	rm -rf docs/*.html

fmt:
	gofmt -w . 