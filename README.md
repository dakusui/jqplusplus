# jq++

A project that productizes the [jq-front](https://github.com/dakusui/jq-front).
jq-front is a simple utility that extends JSON files with file- and node-level inheritance, templating, and reference evaluation, making it easier to create reusable and dynamic JSON-based configurations.

Currently `jq++` supports only inheritance features.
This project follows the standard Go project layout.

## Installation

### Prerequisites

- Go 1.24.5 or later (see [go.mod](go.mod) for the exact version requirement)
- Make (optional, for using the Makefile)

### Install from Source

The easiest way to install `jqplusplus` is using `go install`:

```sh
go install github.com/dakusui/jqplusplus/cmd/jqplusplus@latest
[[ ! -e "$(dirname "$(which jqplusplus)")/jq++" ]] && ln -s "$(which jqplusplus)" "$(dirname "$(which jqplusplus)")/jq++"
```

This will install the `jqplusplus` binary to `$GOPATH/bin` or `$GOBIN` (if set). Make sure this directory is in your `PATH`.

### Build from Source

Clone the repository:

```sh
git clone https://github.com/dakusui/jqplusplus.git
cd jqplusplus
```

Then build using Make:

```sh
make build
```

This will create the executable as `bin/jq++`.

Alternatively, build directly with Go:

```sh
go build -o bin/jq++ ./cmd/jqplusplus
```

### Adding to PATH

After building, you can add the `bin` directory to your PATH, or copy the binary to a directory already in your PATH:

```sh
# Option 1: Add bin directory to PATH (add to ~/.bashrc, ~/.zshrc, etc.)
export PATH="$PATH:$(pwd)/bin"

# Option 2: Copy to a directory in PATH (e.g., /usr/local/bin)
sudo cp bin/jq++ /usr/local/bin/
```

## Usage Example


Let's prepare files, `name.json` and `greeting.json`, from which you want to create a new JSON by extending them.

```shell script
$ echo '{"yourname":"Mark"}' > name.json
$ cat name.json
{"yourname":"Mark"}

$ echo '{"greeting":"Hello"}' > greeting.json
$ cat greeting.json
{"greeting":"Hello"}
```

Then create a file that extends them.

```shell script
$ echo '{
    "$extends": ["greeting.json", "name.json"],
    "sayHello": "eval:$(ref .greeting), $(ref .yourname). Toady is $(date). How are you doing?"
  }' > sayHello.json
```

Now, let's try jq++.
```shell script
$ jq++ sayHello.json
{
  "yourname": "Mark",
  "greeting": "Hello",
  "sayHello": "Hello, Mark. Toady is Fri Aug 30 22:04:40 UTC 2019. How are you doing?"
}
$
```
Doesn't it seem useful? Have fun!

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
## Project Dependencies

* `xmllint`: On Ubuntu, do `sudo apt install libxml2-utils`