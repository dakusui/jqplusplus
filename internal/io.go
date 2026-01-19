package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaterializeLocalNodes materializes Obj["$local"] into files under dir.
// Returns dir (absolute) on success.
func MaterializeLocalNodes(obj map[string]any, localNodeDirectoryBase string) (string, error) {
	if obj == nil {
		return "", errors.New("Obj is nil")
	}
	if strings.TrimSpace(localNodeDirectoryBase) == "" {
		return "", errors.New("dir is empty")
	}

	localAny, ok := obj["$local"]
	if !ok || localAny == nil {
		return "", nil
	}

	localObj, ok := localAny.(map[string]any)
	if !ok {
		return "", fmt.Errorf(`"$local" must be an object (map[string]any), got %T`, localAny)
	}

	absDir, err := os.MkdirTemp(localNodeDirectoryBase, "localnodes-")
	if err != nil {
		return "", fmt.Errorf("mkdir temp dir: %w", err)
	}

	for name, v := range localObj {
		rel, err := sanitizeRelativePath(name)
		if err != nil {
			return "", fmt.Errorf("invalid $local key %q: %w", name, err)
		}

		target := filepath.Join(absDir, rel)

		// Final guard: ensure the resulting path stays within absDir
		relToBase, err := filepath.Rel(absDir, target)
		if err != nil {
			return "", fmt.Errorf("rel check for %q: %w", target, err)
		}
		if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path traversal detected for %q", name)
		}

		// Create parent dirs
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("mkdir parent for %q: %w", target, err)
		}

		data, err := toFileBytes(v)
		if err != nil {
			return "", fmt.Errorf("convert content for %q: %w", name, err)
		}

		// Write file (0644); overwrite if exists
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return "", fmt.Errorf("write %q: %w", target, err)
		}
	}

	return absDir, nil
}

func sanitizeRelativePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty filename")
	}

	// Clean, and normalize separators via filepath.Clean later.
	// Reject absolute paths (Unix and Windows forms).
	if filepath.IsAbs(p) {
		return "", errors.New("absolute paths are not allowed")
	}
	if vol := filepath.VolumeName(p); vol != "" {
		// e.g. "C:" on Windows
		return "", errors.New("volume paths are not allowed")
	}

	clean := filepath.Clean(p)

	// filepath.Clean can turn "." into ".", reject it as "no file".
	if clean == "." {
		return "", errors.New("invalid filename")
	}

	// Reject any traversal.
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}

	// Optional: reject paths containing NUL (can be problematic in some contexts)
	if strings.ContainsRune(clean, '\x00') {
		return "", errors.New("NUL byte in path")
	}

	return clean, nil
}

func toFileBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte(""), nil
	case []byte:
		return x, nil
	case string:
		return []byte(x), nil
	default:
		// JSON-encode other values (maps, arrays, numbers, bools, etc.)
		b, err := json.MarshalIndent(x, "", "  ")
		if err != nil {
			return nil, err
		}
		// Add newline for nicer files
		return append(b, '\n'), nil
	}
}

func SearchPaths() []string {
	v := os.Getenv("JF_PATH")
	return strings.Split(v, ":")
}

func CreateSessionDirectory() string {
	v, ok := os.LookupEnv("JF_SESSION_DIR_BASE")
	if !ok {
		v = ""
	}
	ret, e := os.MkdirTemp(v, "jq++-session-*")
	if e != nil {
		panic(fmt.Sprintf("failed to create session directory: %v", e))
	}
	return ret
}

// ResolveFilePath finds the full path of a referenced file from a list of directories.
// This function works in the following way:
// 1. Iterate over the search paths
//  1. Check if the path exists.
//  2. If it exists, check if it is a true file.
//  3. If it is a true file, return it.
//  4. If it is a directory, return an error.
//
// 2. If the file is not found, return an error.
func ResolveFilePath(filename string, baseDir string, searchPaths []string) (string, string, error) {
	if filepath.IsAbs(filename) {
		return filename, filepath.Dir(filename), nil
	}
	beginning := 0
	if baseDir != "" {
		beginning = -1
	}
	// Iterate over the search paths
	for i := beginning; i < len(searchPaths); i++ {
		var path string
		if i == -1 {
			path = baseDir
		} else {
			path = searchPaths[i]
		}

		// Check if the path exists.
		// 	If exists, return it.
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			// If it is a true file, return it.
			if !os.IsNotExist(err) {
				return fullPath, filepath.Dir(fullPath), nil
			}
			// If it is a directory, return an error.
			return "", "", fmt.Errorf("file is a directory: %s", fullPath)
		}
	}
	return "", "", fmt.Errorf("file not found: %s", filename)
}
