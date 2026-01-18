package internal

import (
	"github.com/itchyny/gojq"
	"testing"
)

/*
Add your test function will be here
*/

func TestInvocationSpecBuilder(t *testing.T) {
	builder := NewInvocationSpecBuilder()

	// Add test module
	builder.AddModules(gojq.WithModuleLoader(gojq.NewModuleLoader(nil)))

	// Add test variable
	builder.AddVariable("var1", "value1")

	t.Log("TestInvocationSpecBuilder passed")
}
