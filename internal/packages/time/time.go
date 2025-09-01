package time

import (
	"time"

	"github.com/araddon/dateparse"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var timeStruct *language.Struct

func NewTime(dg *debug.Debug) language.Object {
	if timeStruct == nil {
		timeStruct = language.NewStruct("time", []language.StructField{
			{Name: "unix", Type: language.TypeInt},
		}, dg)
	}

	t := n.NewPackage("time", dg)
	proto := t.GetPrototype()

	proto.SetObject("Time", timeStruct)
	proto.SetObject("now", n.Function(n.Describe().Returns(timeStruct.Type()), fnNow))
	proto.SetObject("parse", n.Function(n.Describe(n.Arg("format", n.TString), n.Arg("value", n.TString)).Returns(timeStruct.Type()), fnParse))
	proto.SetObject("parseAny", n.Function(n.Describe(n.Arg("time", n.TUnion(n.TString, n.TInt))).Returns(timeStruct.Type()), fnFrom))

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
