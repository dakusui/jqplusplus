package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/nodelevel"
	"log"
	"os"
)

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "Path to the JSON file containing the node content.")
	outputFile := flag.String("output", "", "Path to the output file where the expanded content will be saved.")

	flag.Parse()

	// Validate required flags
	if *inputFile == "" {
		fmt.Println("Error: The '--input' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	// Read the input JSON file
	content, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Parse the JSON file into a map
	var nodeContent map[string]interface{}
	if err := json.Unmarshal(content, &nodeContent); err != nil {
		log.Fatalf("Failed to parse input JSON: %v", err)
	}

	// Call the function to expand node-level filelevel
	expandedContent, err := nodelevel.ExpandNodeLevelInheritances(nodeContent, nodelevel.FetchParentNodeMock)
	if err != nil {
		log.Fatalf("Failed to expand node-level filelevel: %v", err)
	}

	// Serialize expanded content to JSON
	expandedJSON, err := json.MarshalIndent(expandedContent, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize expanded content: %v", err)
	}

	// Save the expanded content to a file or print it to stdout
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, expandedJSON, 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Expanded content saved to: %s\n", *outputFile)
	} else {
		fmt.Println("Expanded Content:")
		fmt.Println(string(expandedJSON))
	}
}
