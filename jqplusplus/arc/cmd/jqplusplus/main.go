package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/arc/internal/filelevel"
	"log"
	"os"

	"github.com/itchyny/gojq"
)

func runGoJQDemo() {
	query, err := gojq.Parse(".foo | .bar")
	if err != nil {
		log.Fatalf("Failed to parse jq query: %v", err)
	}
	input := map[string]interface{}{"foo": map[string]interface{}{"bar": 42}}
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalf("jq error: %v", err)
		}
		fmt.Printf("gojq result: %#v\n", v)
	}
}

func main() {
	versionFlag := flag.Bool("version", false, "Print the version and exit")
	inputFile := flag.String("input", "", "Input JSON file to process filelevel")
	flag.Parse()

	if *versionFlag {
		fmt.Println("jq++ version 0.1.0")
		os.Exit(0)
	}

	if *inputFile != "" {
		result, err := filelevel.LoadAndResolve(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		return
	}

	fmt.Println("Hello from jq++ CLI!")
	runGoJQDemo()
}
