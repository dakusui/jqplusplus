# jqplusplus

This project follows the standard Go project layout.

## Project Structure

- `cmd/jqplusplus/main.go`: Application entry point
- `internal/`: Private application and library code
- `pkg/`: Public libraries (if any)
- `go.mod`, `LICENSE`, `README.md`, `Makefile`: Project metadata and configuration

## Building and Running

To build the main application:

```sh
make build
```

This will create the executable as `bin/jq++`.

To run the program:

```sh
./bin/jq++
```

Or run directly without building:

```sh
make run
``` 