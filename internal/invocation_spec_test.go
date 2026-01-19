package internal

import (
	"testing"
)

/*
Add your test function will be here
*/

func TestInvocationSpecBuilder(t *testing.T) {
	builder := NewInvocationSpecBuilder()

	// Add test module
	builder.AddModules(&JqModule{
		Name:           "",
		CompilerOption: nil,
	})

	// Add test variable
	builder.AddVariable("var1", "value1")

	t.Log("TestInvocationSpecBuilder passed")
}
