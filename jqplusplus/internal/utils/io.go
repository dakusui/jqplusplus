package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml" // go get sigs.k8s.io/yaml
)

// FindFileInPath searches rel (relative path) under each base directory in paths, in order.
// Returns the absolute path of the first regular file that exists.
func FindFileInPath(file string, paths []string) (string, error) {
	if file == "" {
		return "", errors.New("empty file")
	}
	// If file is absolute, just check it directly.
	if filepath.IsAbs(file) {
		if isRegularFile(file) {
			return filepath.Clean(file), nil
		}
		return "", os.ErrNotExist
	}

	for _, base := range paths {
		full := filepath.Join(base, file)
		if isRegularFile(full) {
			abs, err := filepath.Abs(full)
			if err != nil {
				return filepath.Clean(full), nil // fallback: return cleaned path
			}
			return abs, nil
		}
	}
	return "", os.ErrNotExist
}

func isRegularFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// Decoder turns bytes into a JSON-compatible Go value (we'll enforce object at the end).
type Decoder interface {
	Decode(io.Reader) (any, error)
	Name() string
}

/* -------------------- JSON -------------------- */

type JSONDecoder struct{ useNumber bool }

func (d JSONDecoder) Name() string { return "json" }

func (d JSONDecoder) Decode(r io.Reader) (any, error) {
	dec := json.NewDecoder(r)
	if d.useNumber {
		dec.UseNumber()
	}
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

/* -------------------- YAML -------------------- */

// YAMLDecoder converts YAML → JSON bytes → reuses JSON decoder (keeps behavior consistent).
type YAMLDecoder struct{ useNumber bool }

func (d YAMLDecoder) Name() string { return "yaml" }

func (d YAMLDecoder) Decode(r io.Reader) (any, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	jb, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return nil, err
	}
	return JSONDecoder{useNumber: d.useNumber}.Decode(bytes.NewReader(jb))
}

/* -------------------- Public API -------------------- */

// Registry of decoder factories
var decoderRegistry = map[string]func(args []string) Decoder{}

// RegisterDecoder allows registration of a decoder constructor
func RegisterDecoder(name string, ctor func(args []string) Decoder) {
	decoderRegistry[name] = ctor
}

// ToDecoder : Factory method that converts NodeUnit into a Decoder
func ToDecoder(n NodeUnit) (Decoder, error) {
	ctor, ok := decoderRegistry[n.decoder]
	if !ok {
		return nil, fmt.Errorf("unknown decoder: %s", n.decoder)
	}
	return ctor(n.args), nil
}

// ReadFileAsJSONObject reads a file using the given Decoder and enforces that the root is an object.
func ReadFileAsJSONObject(filename string, dec Decoder) (map[string]any, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	v, err := dec.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s decode failed: %w", dec.Name(), err)
	}
	obj, ok := v.(map[string]any)
	if !ok || obj == nil {
		return nil, fmt.Errorf("root must be a JSON object (got %T) when using %s", v, dec.Name())
	}
	return obj, nil
}

// AutoReadFileAsJSONObject picks a decoder by extension (json|yaml|yml). Defaults to JSON.
func AutoReadFileAsJSONObject(filename string, useNumber bool) (map[string]any, error) {
	switch ext := filepath.Ext(filename); ext {
	case ".yaml", ".yml":
		return ReadFileAsJSONObject(filename, YAMLDecoder{useNumber: useNumber})
	default: // ".json" or anything else → try JSON
		return ReadFileAsJSONObject(filename, JSONDecoder{useNumber: useNumber})
	}
}

func CreateDecoder(decoderName string, args []string) (Decoder, error) {
	switch decoderName {
	case "json", "default":
		return JSONDecoder{}, nil
	case "yaml":
		return YAMLDecoder{}, nil
	default:
		return nil, fmt.Errorf("unknown decoder: %s%s", decoderName, args)
	}
}
