package time

import (
	"context"
	"time"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

const defaultTimeFormat = "2006-01-02T15:04:05-07:00"

func NewInstance(t time.Time) (language.Object, error) {
	inst, err := timeStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	reCalc := func() {
		ob, _ := inst.GetPrototype().GetObject(ctx, "unix")
		unix := ob.Value().(int64)
		if unix != t.UnixNano() {
			t = time.Unix(0, unix)
		}
	}

	proto := inst.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	proto.SetObject(ctx, "__value__", n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
		reCalc()
		return n.Int64(t.UnixNano(), inst.Debug()), nil
	}))
	proto.SetObject(ctx, "unix", n.Int64(t.UnixNano(), inst.Debug()))
	proto.SetObject(ctx, "string", n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
		reCalc()
		return n.String(t.Format(FormatTime(defaultTimeFormat)), inst.Debug()), nil
	}))
	proto.SetObject(ctx, "format", n.Function(n.Describe(n.Arg("format", n.TString, n.String(defaultTimeFormat, inst.Debug()))).Returns(n.TString), func(a *n.Args) (any, error) {
		reCalc()
		format := FormatTime(a.Name("format").String())
		return n.String(t.Format(format), inst.Debug()), nil
	}))

	proto.SetObject(ctx, "add", n.Function(n.Describe(n.Arg("time", n.TUnion(inst.Type(), n.TInt))).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		var unix int64

		obj := a.Name("time")
		if n.TInt.Compare(obj.Type()) {
			unix = obj.Value().(int64)
		} else if inst.Type().Compare(obj.Type()) {
			val, _ := inst.GetPrototype().GetObject(ctx, "unix")
			unix = val.Value().(int64)
		}

		return NewInstance(t.Add(time.Duration(unix) * time.Second))
	}))

	proto.SetObject(ctx, "substract", n.Function(n.Describe(n.Arg("time", n.TUnion(inst.Type(), n.TInt))).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		var unix int64

		obj := a.Name("time")
		if n.TInt.Compare(obj.Type()) {
			unix = obj.Value().(int64)
		} else if inst.Type().Compare(obj.Type()) {
			val, _ := inst.GetPrototype().GetObject(ctx, "unix")
			unix = val.Value().(int64)
		}

		return NewInstance(t.Add(-time.Second * time.Duration(unix)))
	}))

	proto.SetObject(ctx, "addDays", n.Function(n.Describe(n.Arg("days", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		days := int(a.Name("days").Value().(int64))
		return NewInstance(t.AddDate(0, 0, days))
	}))

	proto.SetObject(ctx, "addMonths", n.Function(n.Describe(n.Arg("months", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		months := int(a.Name("months").Value().(int64))
		return NewInstance(t.AddDate(0, months, 0))
	}))

	proto.SetObject(ctx, "addYears", n.Function(n.Describe(n.Arg("years", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		years := int(a.Name("years").Value().(int64))
		return NewInstance(t.AddDate(years, 0, 0))
	}))

	proto.SetObject(ctx, "year", intFn(func() int {
		reCalc()
		return t.Year()
	}))
	proto.SetObject(ctx, "month", intFn(func() int {
		reCalc()
		return int(t.Month())
	}))
	proto.SetObject(ctx, "day", intFn(func() int {
		reCalc()
		return t.Day()
	}))
	proto.SetObject(ctx, "hour", intFn(func() int {
		reCalc()
		return t.Hour()
	}))
	proto.SetObject(ctx, "minute", intFn(func() int {
		reCalc()
		return t.Minute()
	}))
	proto.SetObject(ctx, "second", intFn(func() int {
		reCalc()
		return t.Second()
	}))

	return inst, nil
}

func intFn(fn func() int) language.Object {
	return n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
		return n.Int(fn(), nil), nil
	})
}
