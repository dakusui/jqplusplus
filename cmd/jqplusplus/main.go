package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/dakusui/jqplusplus/internal"
)

var version = "dev"      // overridden via -ldflags
var revision = "unknown" // overridden via -ldflags

// versionStrings returns (version, revision) for display. When the binary was
// built with "go install ...@version" (so version/revision are still "dev"/"unknown"),
// the module version is read from the Go build info.
func versionStrings() (string, string) {
	v, r := version, revision
	if v == "dev" || r == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
			// Use first 7 chars of VCS revision from build settings if available
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && s.Value != "" {
					rev := s.Value
					if len(rev) > 7 {
						rev = rev[:7]
					}
					r = rev
					break
				}
			}
		}
	}
	return v, r
}

func main() {
	// Handle symbolic link creation when invoked as jqplusplus with no arguments
	if len(os.Args) == 1 {
		execPath, err := os.Executable()
		if err == nil && strings.HasSuffix(filepath.Base(execPath), "jqplusplus") {
			absExecPath, err := filepath.Abs(execPath)
			if err != nil {
				panic(err)
			}
			err = handleSymbolicLink(absExecPath)
			if err != nil {
				panic(err)
			}
			return
		}
	}

	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		help := `Usage: <program> [options] [files...]

Options:
  -h, --help          Show this help message
  -v, --version       Show version info
  -t, --input <type>  Treat stdin as <type> (e.g. yaml, json, toml, json5,
                      hocon, and their ++ variants). Only applies when reading
                      from stdin; input from files is detected by extension.

If no files are provided, input is read from stdin (JSON by default).
`
		_, _ = os.Stderr.WriteString(help)
		os.Exit(0)
	}

	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		v, r := versionStrings()
		if r == "unknown" {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("jq++ version %v\n", v))
		} else {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("jq++ version %v, build %v\n", v, r))
		}
		os.Exit(0)
	}

	files, inputExt, err := parseArgs(os.Args[1:])
	if err != nil {
		_, _ = os.Stderr.WriteString("Error processing arguments: " + err.Error() + "\n")
		os.Exit(1)
	}

	in, d, err := inputFiles(files, inputExt)
	defer d()
	if err != nil {
		_, _ = os.Stderr.WriteString("Error processing arguments: " + err.Error() + "\n")
		os.Exit(1)
	}
	exitCode := processNodeEntryKeys(in)
	os.Exit(exitCode)
}

// parseArgs splits jq++ operands into input files and an optional stdin input
// type (from -t/--input). The type is normalized to a file-extension suffix
// (e.g. ".yaml"); it is an error to supply it alongside file operands, since
// those are detected by their own extension.
func parseArgs(args []string) (files []string, inputExt string, err error) {
	var inputType string
	haveInputType := false
	setType := func(v string) error {
		ext, ok := internal.InputTypeToExt(v)
		if !ok {
			return fmt.Errorf("unsupported input type %q; supported types are: %s", v, internal.SupportedExtensions)
		}
		inputType, haveInputType = ext, true
		return nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-t" || a == "--input":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("%s requires an argument (e.g. yaml, json, toml)", a)
			}
			i++
			if err := setType(args[i]); err != nil {
				return nil, "", err
			}
		case strings.HasPrefix(a, "--input="):
			if err := setType(strings.TrimPrefix(a, "--input=")); err != nil {
				return nil, "", err
			}
		case strings.HasPrefix(a, "-t="):
			if err := setType(strings.TrimPrefix(a, "-t=")); err != nil {
				return nil, "", err
			}
		default:
			files = append(files, a)
		}
	}
	if haveInputType && len(files) > 0 {
		return nil, "", fmt.Errorf("-t/--input only applies to stdin; input from files is detected by extension")
	}
	return files, inputType, nil
}

func inputFiles(args []string, inputExt string) ([]internal.NodeEntryKey, func(), error) {
	var in []internal.NodeEntryKey
	exit := func() {}
	if len(args) == 0 {
		tempFile, err := os.CreateTemp("", "input-*"+inputExt)
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

		in = []internal.NodeEntryKey{
			internal.NewNodeEntryKey("", absolutePath),
		}
	} else {
		in = internal.Map(args, func(t string) internal.NodeEntryKey {
			return internal.NewNodeEntryKey(filepath.Dir(t), filepath.Base(t))
		})
	}
	return in, exit, nil
}

func processNodeEntryKeys(in []internal.NodeEntryKey) int {
	ret := 0
	for _, eachNodeEntryKey := range in {
		v, err := processNodeEntryKey(eachNodeEntryKey)
		if err != nil {
			ret = 1
			_, _ = os.Stderr.WriteString(fmt.Sprintf("ERROR: %s: in %v\n", err.Error(), eachNodeEntryKey))
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

func handleSymbolicLink(execPath string) error {
	execDir := filepath.Dir(execPath)
	symlinkPath := filepath.Join(execDir, "jq++")

	// Check if symlink exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		// Remove existing symlink
		_ = os.Remove(symlinkPath)
	}

	// Create a new symlink
	err := os.Symlink(execPath, symlinkPath)
	return err
}

func processNodeEntryKey(nodeEntryKey internal.NodeEntryKey) (string, error) {
	nodeEntryValue, err := internal.LoadAndResolveInheritances(nodeEntryKey.BaseDir(), nodeEntryKey.Filename(), internal.SearchPaths())
	if err != nil {
		return "", err
	}
	obj := nodeEntryValue.Obj
	// Ground array-composition deltas: tokens that survived every merge can
	// never be resolved and are reported before evaluation begins.
	if err := internal.ValidateArrayComposition(obj); err != nil {
		return "", err
	}
	{
		invocationSpec := internal.NewInvocationSpecBuilder().AddModules(nodeEntryValue.CompilerOptions...).Build()
		obj, err = internal.ProcessKeySide(obj, 7, *invocationSpec, nodeEntryKey.BaseDir(), internal.SearchPaths())
		if err != nil {
			return "", err
		}
	}
	{
		invocationSpec := internal.NewInvocationSpecBuilder().AddModules(nodeEntryValue.CompilerOptions...).Build()
		obj, err = internal.ProcessValueSide(obj, 7, *invocationSpec, nodeEntryKey.BaseDir(), internal.SearchPaths())
		if err != nil {
			return "", err
		}
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
