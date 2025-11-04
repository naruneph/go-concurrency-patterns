package registry

import (
	"fmt"
	"sort"
)

type Example struct {
	Name string
	Doc  string
	Func func()
}

// Examples is a registry of concurrency examples.
var Examples = map[string]Example{}

// Register adds a new concurrency example to the registry.
func Register(name string, doc string, example func()) {
	if _, exists := Examples[name]; exists {
		panic(fmt.Sprintf("example %q already registered", name))
	}
	Examples[name] = Example{
		Name: name,
		Doc:  doc,
		Func: example,
	}
}

// List returns a sorted list of registered concurrency example names.
func List() []string {
	names := []string{}
	for k := range Examples {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
