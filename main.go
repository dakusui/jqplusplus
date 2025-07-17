package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println("jqplusplus version 0.1.0")
		os.Exit(0)
	}

	fmt.Println("Hello from jqplusplus CLI!")
}
