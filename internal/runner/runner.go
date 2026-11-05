package runner

import (
	"os"
	"time"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/language"
)

func Execute(path string, r *runtime.Runtime, parseTime bool) (result language.Object, duration *time.Duration, err error) {
	defer func() {
		if os.Getenv("NUBO_DEV") != "true" {
			if recovered := recover(); recovered != nil {
				result = nil
				err = exception.Create("Language specific error occured")
			}
		}
	}()

	var start time.Time
	if parseTime {
		start = time.Now()
	}

	var nodes []*astnode.Node
	if r.Source().Disk() {
		nodes, err = parseFile(path)
	} else {
		path, err = r.Source().Resolve(".", path)
		if err == nil {
			nodes, err = r.Source().Nodes(path)
		}
	}

	if parseTime {
		elapsed := time.Since(start)
		duration = &elapsed
	}

	if err != nil {
		return nil, duration, err
	}

	result, err = r.Interpret(path, nodes)
	if err != nil {
		return nil, duration, err
	}

	return result, duration, nil
}
