package time

import (
	"context"
	"fmt"
	"time"

	"github.com/araddon/dateparse"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var monotonicOrigin = time.Now()

var timeStruct *language.Struct

func GetInstance() *language.Struct {
	if timeStruct == nil {
		timeStruct = language.NewStruct("time", []language.StructField{
			{Name: "unix", Type: language.TypeInt, Private: true},
		}, nil)
	}

	return timeStruct
}

func NewTime(dg *debug.Debug) language.Object {
	GetInstance()
	t := n.NewPackage("time", dg)
	proto := t.GetPrototype()
	ctx := context.Background()

	proto.SetObject(ctx, "Time", timeStruct)
	proto.SetObject(ctx, "now", n.Function(n.Describe().Returns(timeStruct.Type()), fnNow))
	proto.SetObject(ctx, "parse", n.Function(n.Describe(n.Arg("format", n.TString), n.Arg("value", n.TString)).Returns(timeStruct.Type()), fnParse))
	proto.SetObject(ctx, "parseAny", n.Function(n.Describe(n.Arg("time", n.TUnion(n.TString, n.TInt))).Returns(timeStruct.Type()), fnFrom))

	proto.SetObject(ctx, "monotonicNs", n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) { return time.Since(monotonicOrigin).Nanoseconds(), nil }))
	for _, name := range []string{"sleepNs", "sleepUntil"} {
		argument := "duration"
		if name == "sleepUntil" {
			argument = "deadline"
		}
		proto.SetObject(ctx, name, n.Function(n.Describe(n.Arg(argument, n.TInt)), func(a *n.Args) (any, error) {
			ns := a.Get(0).Value().(int64)
			if ns < 0 {
				return nil, fmt.Errorf("%s requires nonnegative nanoseconds", name)
			}
			if name == "sleepUntil" {
				ns -= time.Since(monotonicOrigin).Nanoseconds()
			}
			if ns > 0 {
				time.Sleep(time.Duration(ns))
			}
			return nil, nil
		}))
	}
	return t
}

func fnNow(args *n.Args) (any, error) {
	return NewInstance(time.Now())
}

func fnParse(args *n.Args) (any, error) {
	t, err := time.Parse(FormatTime(args.Name("format").String()), args.Name("value").String())
	if err != nil {
		return nil, err
	}
	return NewInstance(t)
}

func fnFrom(args *n.Args) (any, error) {
	tim := args.Name("time")
	if tim.Type().Base() == language.ObjectTypeInt {
		return NewInstance(time.Unix(tim.Value().(int64), 0))
	}

	t, err := dateparse.ParseAny(tim.String())
	if err != nil {
		return nil, err
	}
	return NewInstance(t)
}
