package main

import (
	"github.com/dakusui/jqplusplus/internal"
	"os"
)

func main() {
	internal.LoadAndResolveInheritances("", "", os.Args[1:])
	os.Exit(0)
}
