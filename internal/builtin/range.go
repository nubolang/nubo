package builtin

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/packages/iter"
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

var rangeStruct *language.Struct

func getRangeStruct() language.Object {
	ctx := context.Background()

	if rangeStruct == nil {
		rangeStruct = language.NewStruct("range", []language.StructField{
			{Name: "start", Type: n.TInt},
			{Name: "stop", Type: n.TInt},
			{Name: "step", Type: n.TInt},
		}, nil)

		sp := rangeStruct.GetPrototype().(*language.StructPrototype)

		sp.Unlock()

		sp.SetObject(ctx, "init", n.Function(n.Describe(
			n.Arg("self", rangeStruct.Type()),
			n.Arg("start", n.TInt),
			n.Arg("stop", n.Nullable(n.TInt), language.Nil),
			n.Arg("step", n.TInt, n.Int(1)),
		).Returns(rangeStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				proto := self.GetPrototype()
				ctx := language.StructAllowPrivateCtx(context.Background())

				startObj := a.Name("start")
				stopObj := a.Name("stop")
				stepObj := a.Name("step")

				var (
					start int
					step  int
					stop  int
				)

				start = int(startObj.(*language.Int).Value().(int64))
				step = int(stepObj.(*language.Int).Value().(int64))

				if step == 0 {
					return nil, exception.Create("range: step cannot be 0").WithDebug(a.Name("step").Debug()).WithLevel(exception.LevelRuntime)
				}

				if stopObj == nil || stopObj.Value() == nil {
					stop = start
					start = 0
				} else {
					stop = int(stopObj.(*language.Int).Value().(int64))
				}

				proto.SetObject(ctx, "start", n.Int(start, startObj.Debug()))
				proto.SetObject(ctx, "stop", n.Int(stop, stopObj.Debug()))
				proto.SetObject(ctx, "step", n.Int(step, stepObj.Debug()))

				return self, nil
			}))

		it := iter.NewIter(nil)
		proto := it.GetPrototype()
		iterator, _ := proto.GetObject(ctx, "Iterator")
		end, _ := proto.GetObject(ctx, "End")
		progress, _ := proto.GetObject(ctx, "Progress")

		sp.SetObject(ctx, "__iterate__", n.Function(n.Describe(n.Arg("self", rangeStruct.Type())).Returns(iterator.Type()), func(a *n.Args) (any, error) {
			self := a.Name("self")
			proto := self.GetPrototype()
			ctx := language.StructAllowPrivateCtx(context.Background())

			interInst, err := iterator.(*language.Struct).NewInstance()
			if err != nil {
				return nil, err
			}

			iterProto := interInst.GetPrototype()
			iterInit, _ := iterProto.GetObject(ctx, "init")

			fn := iterInit.(*language.Function)

			startObj, _ := proto.GetObject(ctx, "start")
			stopObj, _ := proto.GetObject(ctx, "stop")
			stepObj, _ := proto.GetObject(ctx, "step")

			start := int(startObj.(*language.Int).Value().(int64))
			stop := int(stopObj.(*language.Int).Value().(int64))
			step := int(stepObj.(*language.Int).Value().(int64))
			current := start

			return fn.Data(ctx, []language.Object{
				n.Function(n.Describe().Returns(progress.Type()), func(a *n.Args) (any, error) {
					if (step > 0 && current >= stop) || (step < 0 && current <= stop) {
						return end, nil
					}

					defer func() { current += step }()

					inst, _ := progress.(*language.Struct).NewInstance()
					prog := inst.GetPrototype()

					progInit, _ := prog.GetObject(ctx, "init")

					return progInit.(*language.Function).Data(ctx, []language.Object{
						n.Int(current),
						n.Int(current),
					})
				}),
			})
		}))

		sp.Lock()
		sp.Implement()
	}

	return rangeStruct
}
