package internal

import (
	"github.com/itchyny/gojq"
	"sort"
)

// InvocationSpec represents a specification for invoking a functionality
// with a set of modules and variables.
//
// This type should be constructed using the InvocationSpecBuilder.
//
// Fields:
//   - modules: A slice of gojq.CompilerOption values representing modules
//     required for invocation. These could modify or enrich the behavior of the compiler.
//   - variables: A map where the keys are variable names (strings) and
//     values are of type any, representing the parameters for the invocation.
type InvocationSpec struct {
	modules   []gojq.CompilerOption
	variables map[string]any
}

// VariableNames returns a slice of all variable names present in the InvocationSpec.
// The returned keys are sorted in a dictionary order.
//
// Returns:
// - []string: A slice containing all the keys (variable names) in the `variables` map.
func (spec *InvocationSpec) VariableNames() []string {
	if spec.variables == nil {
		spec.variables = map[string]any{}
	}

	keys := make([]string, 0, len(spec.variables))
	for key := range spec.variables {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// ValueOf retrieves the value associated with the given variable name from the InvocationSpec's variables map.
// This method should be used for a variable name returned by VariableNames method.
//
// Panics:
// - If the `variableName` does not exist in the `variables` map, a panic is raised with the message "variable not found".
//
// Parameters:
// - variableName: The name of the variable whose value is to be retrieved.
//
// Returns:
// - any: The value associated with the specified variable name.
func (spec *InvocationSpec) ValueOf(variableName string) any {
	if spec.variables == nil {
		spec.variables = map[string]any{}
	}
	if _, ok := spec.variables[variableName]; !ok {
		panic("variable not found")
	}
	return spec.variables[variableName]
}

// VariableValues returns a slice containing all the values of the variables
// present in the InvocationSpec. The values are ordered according to the
// alphabetical order of their corresponding variable names.
//
// Returns:
// - []any: A slice of variable values sorted in the order of their keys.
func (spec *InvocationSpec) VariableValues() []any {
	ret := make([]any, len(spec.variables))
	i := 0
	for _, v := range spec.VariableNames() {
		ret[i] = spec.ValueOf(v)
		i++
	}
	return ret
}

func (spec *InvocationSpec) Modules() []gojq.CompilerOption {
	if spec.modules == nil {
		spec.modules = make([]gojq.CompilerOption, 0)
	}
	return spec.modules
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

// FromSpec creates a new InvocationSpecBuilder initialized with an existing InvocationSpec.
//
// Parameters:
// - spec: An instance of InvocationSpec to initialize the builder with.
//
// Returns:
// - *InvocationSpecBuilder: An instance of InvocationSpecBuilder initialized with the provided spec.
func FromSpec(spec *InvocationSpec) *InvocationSpecBuilder {
	return &InvocationSpecBuilder{
		spec: &InvocationSpec{
			modules: append([]gojq.CompilerOption{}, spec.modules...),
			variables: func() map[string]any {
				cloned := map[string]any{}
				for k, v := range spec.variables {
					cloned[k] = v
				}
				return cloned
			}(),
		},
	}
}

func (b *InvocationSpecBuilder) AddModules(modules ...gojq.CompilerOption) *InvocationSpecBuilder {
	b.spec.modules = append(b.spec.modules, modules...)
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
