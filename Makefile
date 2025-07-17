APP_NAME=jq++
SRC=main.go

.PHONY: all build run clean

all: build

build:
	go build -o $(APP_NAME) $(SRC)

run: build
	./$(APP_NAME)

clean:
	rm -f $(APP_NAME) 