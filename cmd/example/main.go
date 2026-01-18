package main

import (
	"fmt"
	"log"

	"github.com/itchyny/gojq"
)

// ---- Module loader ----

type moduleLoader struct {
	moduleName string
	moduleBody string
}

func (l *moduleLoader) LoadModule(name string) (*gojq.Query, error) {
	if l.moduleName == "mymod" {
		return gojq.Parse(fmt.Sprintf(`%s`, l.moduleBody))
	}
	return nil, fmt.Errorf("module not found: %s", name)
}

func newModuleLoader(moduleName, moduleBody string) *moduleLoader {
	return &moduleLoader{moduleName, moduleBody}
}

func main() {
	// ---- Main jq query ----
	query, err := gojq.Parse(`
import "mymod" as m;
m::custom_func
`)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	// Compile
	code, err := gojq.Compile(
		query,
		gojq.WithModuleLoader(newModuleLoader("mymod", `
def custom_func:
  { new_key: . };
`)),
	)
	if err != nil {
		log.Fatalf("compile error: %v", err)
	}

	// Run
	iter := code.Run("hello")

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalf("runtime error: %v", err)
		}
		fmt.Printf(">%#v\n", v)
	}
}
