package runner

import (
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/language"
)

func Execute(path string, r *runtime.Runtime) (language.Object, error) {
	nodes, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	return r.Interpret(path, nodes)
}
