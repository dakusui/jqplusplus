package internal

import (
	"fmt"
	"github.com/itchyny/gojq"
	"testing"
)

/*
Add your test function will be here
*/

func TestInvocationSpecBuilder(t *testing.T) {
	builder := NewInvocationSpecBuilder()

	// Add test module
	builder.AddModule(gojq.WithModuleLoader(gojq.NewModuleLoader(nil)))

	// Add test variable
	builder.AddVariable("var1", "value1")

	fmt.Println(builder.Build())

	t.Log("TestInvocationSpecBuilder passed")
}
