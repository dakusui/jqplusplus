package internal

import "github.com/itchyny/gojq"

type InvocationSpec struct {
	modules   []gojq.CompilerOption
	variables map[string]any
}

type InvocationSpecBuilder struct {
	spec *InvocationSpec
}

// NewInvocationSpecBuilder creates a new InvocationSpec builder.
func NewInvocationSpecBuilder() *InvocationSpecBuilder {
	return &InvocationSpecBuilder{
		spec: &InvocationSpec{},
	}
}

// AddModule adds a gojq.CompilerOption to the InvocationSpec's modules slice.
func (b *InvocationSpecBuilder) AddModule(module gojq.CompilerOption) *InvocationSpecBuilder {
	b.spec.modules = append(b.spec.modules, module)
	return b
}

// AddVariable adds a variable to the InvocationSpec's variables map.
func (b *InvocationSpecBuilder) AddVariable(name string, value any) *InvocationSpecBuilder {
	if b.spec.variables == nil {
		b.spec.variables = map[string]any{}
	}
	b.spec.variables[name] = value
	return b
}

// Build returns the built InvocationSpec.
func (b *InvocationSpecBuilder) Build() *InvocationSpec {
	return b.spec
}

// Clone creates a deep copy of the existing InvocationSpecBuilder.
// Note: Go does not support cloning of interfaces out of the box,
// the variables and CompilerOptions should be safe to use with the reference semantics.
func (b *InvocationSpecBuilder) Clone() *InvocationSpecBuilder {
	newBuilder := NewInvocationSpecBuilder()
	newBuilder.spec.variables = make(map[string]any, len(b.spec.variables))

	for k, v := range b.spec.variables {
		newBuilder.spec.variables[k] = v
	}

	newBuilder.spec.modules = append(newBuilder.spec.modules, b.spec.modules...)

	return newBuilder
}
