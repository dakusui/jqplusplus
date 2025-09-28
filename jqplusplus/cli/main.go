package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dakusui/jqplusplus/jqplusplus/internal/utils"
	"log"
	"os"
	"path/filepath"
)

func main() {
	err := mainBody()
	if err != nil {
		log.Fatal(err)
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

	// Define CLI flag
	file := flag.String("file", "", "Path to input file (JSON, YAML, etc.)")
	flag.Parse()
	if *file == "" {
		return fmt.Errorf("please provide --file argument")
	}

	// Example: use standard JSON decoder here
	elem, err := utils.ReadFileAsJsonElement(*file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
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

	v, err := utils.AsStringArray(extends, nil)
	if err != nil {
		return err
	}
	paths := []string{filepath.Dir(*file), dir + "/" + "testdata"}
	reverse(v)
	cur := utils.NewEmptyObject()
	for _, each := range v {
		println(each)
		nodeUnit, err := utils.ParseNodeUnit(each)
		if err != nil {
			return err
		}
		obj, err := readFileAsJsonObject(*nodeUnit, paths)
		if err != nil {
			return err
		}
		println("***")
		prettyPrint(obj)
		cur, err = utils.MergeObjectNodes(cur, obj)
		if err != nil {
			return err
		}
	}
	println("----")
	prettyPrint(cur)
	println("----")
	return nil
}

func readFileAsJsonObject(n utils.NodeUnit, paths []string) (map[string]interface{}, error) {
	f, err := utils.FindFileInPath(n.Name(), paths)
	if err != nil {
		return nil, err
	}
	decoder, err := utils.CreateDecoder(n.Decoder(), n.Args())
	if err != nil {
		return nil, err
	}
	ret, err := utils.ReadFileAsJSONObject(f, decoder)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func prettyPrint(obj any) {
	// Pretty print the object
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		log.Fatalf("Failed to print JSON: %v", err)
	}
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
