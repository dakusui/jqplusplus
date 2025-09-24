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
	elem, err := utils.ReadFileAsJsonElement(*file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
		panic(err)
	}

	prettyPrint(elem)
	extends, err := utils.FindByPathExpression(elem, ".\"$extends\"")

	if err != nil {
		switch err.(type) {
		case *utils.JsonNotFound:
			break
		case *utils.JsonTypeError:
		default:
			panic(err)
		}
	}

	prettyPrint(extends)
	v, err := utils.AsStringArray(extends, nil)
	if err != nil {
		panic(err)
	}
	for _, each := range v {
		println(each)
	}

	println(elem)
}

func prettyPrint(obj any) {
	// Pretty print the object
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		log.Fatalf("Failed to print JSON: %v", err)
		panic(err)
	}
}
