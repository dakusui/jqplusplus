package main

import (
	"encoding/json"
	"fmt"
	"github.com/dakusui/jqplusplus/internal"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		help := `Usage: <program> [options] [files...]

Options:
  -h, --help   Show this help message

If no files are provided, input is read from stdin.
`
		_, _ = os.Stdout.WriteString(help)
		os.Exit(0)
	}

	in, d, err := inputFiles(os.Args)
	defer d()
	if err != nil {
		_, _ = os.Stderr.WriteString("Error processing arguments: " + err.Error() + "\n")
		os.Exit(1)
	}
	exitCode := processNodeEntries(in)
	os.Exit(exitCode)
}

func inputFiles(args []string) ([]internal.NodeEntry, func(), error) {
	var in []internal.NodeEntry
	exit := func() {}
	if len(args) == 1 {
		tempFile, err := os.CreateTemp("", "input-*")
		if err != nil {
			_, _ = os.Stderr.WriteString("Error creating temporary file: " + err.Error() + "\n")
			return nil, exit, err
		}
		exit = func() {
			err := os.Remove(tempFile.Name())
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "\n")
			}
		}

		if _, err := tempFile.ReadFrom(os.Stdin); err != nil {
			_, _ = os.Stderr.WriteString("Error reading from stdin: " + err.Error() + "\n")
			return nil, exit, err
		}

		if err := tempFile.Close(); err != nil {
			_, _ = os.Stderr.WriteString("Error closing temporary file: " + err.Error() + "\n")
			return nil, exit, err
		}
		absolutePath, err := filepath.Abs(tempFile.Name())
		if err != nil {
			_, _ = os.Stderr.WriteString("Error getting absolute path: " + err.Error() + "\n")
			return nil, exit, err
		}

		in = []internal.NodeEntry{
			internal.NewNodeEntry("", absolutePath),
		}
	} else {
		in = internal.Map(args[1:], func(t string) internal.NodeEntry {
			return internal.NewNodeEntry(filepath.Dir(t), filepath.Base(t))
		})
	}
	return in, exit, nil
}

func processNodeEntries(in []internal.NodeEntry) int {
	ret := 0
	for _, nodeEntry := range in {
		v, err := processNodeEntry(nodeEntry)
		if err != nil {
			_, _ = os.Stderr.WriteString("Error processing file " + nodeEntry.String() + ": " + err.Error() + "\n")
			ret = 1
			break
		}
		_, err = os.Stdout.WriteString(v + "\n")
		if err != nil {
			ret = 1
			break
		}
	}
	return ret
}

func processNodeEntry(nodeEntry internal.NodeEntry) (string, error) {
	obj, err := internal.LoadAndResolveInheritances(nodeEntry.BaseDir(), nodeEntry.Filename(), internal.SearchPaths())
	if err != nil {
		return "", err
	}
	obj, err = internal.ProcessKeySide(obj, 7, internal.EmptyInvocationSpec())
	if err != nil {
		return "", err
	}
	obj, err = internal.ProcessValueSide(obj, 7, internal.EmptyInvocationSpec())
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
