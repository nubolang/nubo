package time

import (
	"time"

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

	proto.SetObject("time", timeStruct)
	proto.SetObject("now", n.Function(n.Describe().Returns(timeStruct.Type()), fnNow))
	proto.SetObject("parse", n.Function(n.Describe(n.Arg("format", n.TString), n.Arg("value", n.TString)), fnFrom))

	return t
}

func fnNow(args *n.Args) (any, error) {
	return NewInstance(time.Now())
}

func fnFrom(args *n.Args) (any, error) {
	t, err := time.Parse(FormatTime(args.Name("format").String()), args.Name("value").String())
	if err != nil {
		return nil, err
	}
	return NewInstance(t)
}
