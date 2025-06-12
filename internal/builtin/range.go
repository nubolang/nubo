package builtin

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func rangeFn(args *n.Args) (any, error) {
	startObj := args.Name("start")
	stopObj := args.Name("stop")
	stepObj := args.Name("step")

	var (
		start int = int(startObj.Value().(int64))
		stop  int
		step  int = int(stepObj.Value().(int64))
	)

	if step == 0 {
		return nil, debug.NewError(fmt.Errorf("Range error"), "step cannot be 0", stepObj.Debug())
	}

	if stopObj.Type() == n.TNil {
		stop = start
		start = 0
	} else {
		stop = int(stopObj.Value().(int64))
	}

	var result []language.Object
	if step > 0 {
		for i := start; i < stop; i += step {
			result = append(result, n.Int(i, startObj.Debug()))
		}
	} else {
		for i := start; i > stop; i += step {
			result = append(result, n.Int(i, startObj.Debug()))
		}
	}

	return language.NewList(result, n.TInt, startObj.Debug()), nil
}
