package main

import (
	"encoding/json"
	"flag"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"io"
	"log"
	"os"
)

func main() {
	// Define CLI flag
	file := flag.String("file", "", "Path to input file (JSON, YAML, etc.)")
	flag.Parse()

	if *file == "" {
		log.Fatal("Please provide --file argument")
	}

	// Example: use standard JSON decoder here
	obj, err := utils.ReadFileAsObjectNode(*file, func(data []byte) (map[string]any, error) {
		var m map[string]any
		err := json.Unmarshal(data, &m)
		return m, err
	})
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Pretty print the object
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		log.Fatalf("Failed to print JSON: %v", err)
	}
}
