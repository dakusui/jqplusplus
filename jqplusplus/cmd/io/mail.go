package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Assume this is already implemented in your project
func ReadFileAsJsonObject(filename string, decoder func([]byte) (map[string]any, error)) (map[string]any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return decoder(data)
}

func main() {
	// Define CLI flag
	file := flag.String("file", "", "Path to input file (JSON, YAML, etc.)")
	flag.Parse()

	if *file == "" {
		log.Fatal("Please provide --file argument")
	}

	// Example: use standard JSON decoder here
	obj, err := ReadFileAsJsonObject(*file, func(data []byte) (map[string]any, error) {
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
