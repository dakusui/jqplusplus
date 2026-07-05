package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dakusui/jqplusplus/internal"
	"github.com/dakusui/jqplusplus/internal/testutil"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantFiles []string
		wantExt   string
		wantErr   bool
	}{
		{name: "no args", args: nil, wantFiles: nil, wantExt: ""},
		{name: "files only", args: []string{"a.yaml", "b.json"}, wantFiles: []string{"a.yaml", "b.json"}, wantExt: ""},
		{name: "input yaml", args: []string{"--input", "yaml"}, wantFiles: nil, wantExt: ".yaml"},
		{name: "input short", args: []string{"-t", "toml"}, wantFiles: nil, wantExt: ".toml"},
		{name: "input equals", args: []string{"--input=json5"}, wantFiles: nil, wantExt: ".json5"},
		{name: "input dotted plusplus", args: []string{"-t", ".yaml++"}, wantFiles: nil, wantExt: ".yaml++"},
		{name: "missing value", args: []string{"-t"}, wantErr: true},
		{name: "unsupported type", args: []string{"-t", "xml"}, wantErr: true},
		{name: "type with files", args: []string{"-t", "yaml", "a.yaml"}, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files, ext, err := parseArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got files=%v ext=%q", files, ext)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ext != c.wantExt {
				t.Errorf("ext: want %q, got %q", c.wantExt, ext)
			}
			if !reflect.DeepEqual(files, c.wantFiles) {
				t.Errorf("files: want %v, got %v", c.wantFiles, files)
			}
		})
	}
}

func TestLoadAndResolveInheritances_SingleExtendsForJqFile(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.jq",
		`def custom_func:
  { new_key: .store };
`)
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "$extends": ["parent.jq"],
  "store": "Hello",
  "key": "eval:object:parent::custom_func"
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(map[string]any{"key": map[string]any{"new_key": "Hello"}, "store": "Hello"}, "", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
func TestLoadAndResolveInheritances_TagRef(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "thetag": "Hello, mytag",
  "k0": {
    "k1": "eval:string:reftag(\"thetag\")"
  }
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(
		map[string]any{
			"thetag": "Hello, mytag",
			"k0": map[string]any{
				"k1": "Hello, mytag"}},
		"", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", string(expected), result)
	}
}

func TestLoadAndResolveInheritances_ReftagInsideModuleResolvesNestedEvalStrings(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "funcs.jq",
		`def backendRef($version; $port):
  {"name": reftag("metadata").name + "-" + $version, "port": $port};
`)
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "$extends": ["funcs.jq"],
  "_svc": "reviews",
  "metadata": {
    "name": "eval:string:refexpr(\"._svc\")"
  },
  "backendRef": "eval:object:funcs::backendRef(\"v1\"; 9080)"
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(
		map[string]any{
			"_svc": "reviews",
			"metadata": map[string]any{
				"name": "reviews",
			},
			"backendRef": map[string]any{
				"name": "reviews-v1",
				"port": 9080,
			},
		},
		"", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", string(expected), result)
	}
}

func TestLoadAndResolveInheritances_RefToStringFromInsideArray(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "arr": [
    {
      "k": {
        "v": "eval:string:ref(parentof($cur) + [\"content\"]) | tostring",
        "content": {
          "k1": "v1",
          "k2": "v2"
        }
      }
    }
  ]
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(
		map[string]any{
			"arr": []any{
				map[string]any{
					"k": map[string]any{
						"v": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
						"content": map[string]any{
							"k1": "v1",
							"k2": "v2",
						},
					},
				}}},
		"", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", string(expected), result)
	}
}

func TestLoadAndResolveInheritances_CyclicRef_ThenError(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
          "k1": "eval:refexpr(\".k2\")",
          "k2": "eval:refexpr(\".k1\")"
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err == nil {
		t.Fatalf("error expected but got: %v", result)
	}
}

func TestLoadAndResolveInheritances_IncompatibleCallInExpresseion_ThenError(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
	"arr": "eval:array:refexpr()"
  }
`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err == nil {
		t.Fatalf("error expected but got: %v", result)
	}
}

/*
map[string]interface {} [

	"key": map[string]interface {} [
	  "key": *(*interface {})(0xc0000ee6d8), ], ]
*/
func TestLoadAndResolveInheritances_IndirectCyclicRef_ThenPanic(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "a": {
    "key": "eval:object:ref([\"a\"])"
  }
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))
	if err == nil {
		t.Fatalf("error expected but got: %v", result)
	}
}

func TestLoadAndResolveInheritances_MIssingReftAG_ThenError(t *testing.T) {
	dir := t.TempDir()
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "thetag": "Hello, mytag",
  "k0": {
    "k1": "eval:string:reftag(\"missingTag\")"
  }
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err == nil {
		t.Fatalf("error expected but got: %v", result)
	}
}

func TestLoadAndResolveInheritances_ReadfileAsStringOnPath(t *testing.T) {
	dir := t.TempDir()
	_ = testutil.WriteTempJSON(t, dir, "parent.json",
		`"hello"`)
	child := testutil.WriteTempJSON(t, dir, "child.json",
		`{
  "key": "eval:string:readfile(\"parent.json\")"
}`)
	result, err := processNodeEntryKey((internal.NewNodeEntryKey(filepath.Dir(child), filepath.Base(child))))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected, _ := json.MarshalIndent(map[string]any{"key": "hello"}, "", "  ")
	if !reflect.DeepEqual(result, string(expected)) {
		t.Errorf("expected %v, got %v", string(expected), string(result))
	}
}
