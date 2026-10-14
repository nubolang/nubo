package runner

import (
	"time"

	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/language"
)

func Execute(path string, r *runtime.Runtime, parseTime bool) (language.Object, *time.Duration, error) {
	var start time.Time
	if parseTime {
		start = time.Now()
	}

	nodes, err := parseFile(path)

	var duration *time.Duration
	if parseTime {
		elapsed := time.Since(start)
		duration = &elapsed
	}

	if err != nil {
		return nil, duration, err
	}

	result, err := r.Interpret(path, nodes)
	if err != nil {
		return nil, duration, err
	}

	return result, duration, nil
}
