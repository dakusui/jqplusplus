package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"log"
	"os"
)

func main() {
	err := mainBody()
	if err != nil {
		panic(err)
	}
	os.Exit(
		0,
	)
}

func mainBody() error {
	// Initialize NodeLoader
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	nodePool := utils.NewNodePoolWithSimpleLoader([]string{dir + "/" + "testdata"})

	// Define CLI flag
	file := flag.String("file", "", "Path to input file (JSON, YAML, etc.)")
	flag.Parse()
	if *file == "" {
		log.Fatal("Please provide --file argument")
		return fmt.Errorf("Please provide --file argument")
	}

	// Example: use standard JSON decoder here
	elem, err := utils.ReadFileAsJsonElement(*file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
		return err
	}

	prettyPrint(elem)
	extends, err := utils.FindByPathExpression(elem, ".\"$extends\"")

	if err != nil {
		switch err.(type) {
		case *utils.JsonNotFound:
			break
		case *utils.JsonTypeError:
		default:
			return err
		}
	}

	prettyPrint(extends)
	v, err := utils.AsStringArray(extends, nil)
	if err != nil {
		return err
	}
	for _, each := range v {
		println(each)
		nodeUnit, err := utils.ParseNodeUnit(each)
		if err != nil {
			return err
		}
		obj, err := nodePool.GetNode(*nodeUnit, *file)
		if err != nil {
			return err
		}
		println(fmt.Sprintf("<%s>", obj))
	}
	println(elem)
	return nil
}

func prettyPrint(obj any) {
	// Pretty print the object
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		log.Fatalf("Failed to print JSON: %v", err)
	}
}
