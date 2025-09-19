package main

import (
	"encoding/json"
	"flag"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"log"
	"os"
)

func main() {
	// Define CLI flag
	file := flag.String("file", "", "Path to input file (JSON, YAML, etc.)")
	flag.Parse()

	if *file == "" {
		log.Fatal("Please provide --file argument")
		panic("Please provide --file argument")
	}

	// Example: use standard JSON decoder here
	unit := utils.NewNodeUnit(*file, "json", []string{})
	obj, err := utils.ReadFileAsObjectNode(unit)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
		panic(err)
	}

	// Pretty print the object
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		log.Fatalf("Failed to print JSON: %v", err)
		panic(err)
	}
}
