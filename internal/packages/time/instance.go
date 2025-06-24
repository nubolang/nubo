package time

import (
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

	reCalc := func() {
		ob, _ := inst.GetPrototype().GetObject("unix")
		unix := ob.Value().(int64)
		if unix != t.Unix() {
			t = time.Unix(unix, 0)
		}
	}

	proto := inst.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	proto.SetObject("__value__", n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
		reCalc()
		return n.Int64(t.Unix(), inst.Debug()), nil
	}))
	proto.SetObject("unix", n.Int64(t.Unix(), inst.Debug()))
	proto.SetObject("string", n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
		reCalc()
		return n.String(t.Format(FormatTime(defaultTimeFormat)), inst.Debug()), nil
	}))
	proto.SetObject("format", n.Function(n.Describe(n.Arg("format", n.TString, n.String(defaultTimeFormat, inst.Debug()))).Returns(n.TString), func(a *n.Args) (any, error) {
		reCalc()
		format := FormatTime(a.Name("format").String())
		return n.String(t.Format(format), inst.Debug()), nil
	}))

	proto.SetObject("add", n.Function(n.Describe(n.Arg("time", n.TUnion(inst.Type(), n.TInt))).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		var unix int64

		obj := a.Name("time")
		if n.TInt.Compare(obj.Type()) {
			unix = obj.Value().(int64)
		} else if inst.Type().Compare(obj.Type()) {
			val, _ := inst.GetPrototype().GetObject("unix")
			unix = val.Value().(int64)
		}

		return NewInstance(t.Add(time.Duration(unix) * time.Second))
	}))

	proto.SetObject("sub", n.Function(n.Describe(n.Arg("time", n.TUnion(inst.Type(), n.TInt))).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		var unix int64

		obj := a.Name("time")
		if n.TInt.Compare(obj.Type()) {
			unix = obj.Value().(int64)
		} else if inst.Type().Compare(obj.Type()) {
			val, _ := inst.GetPrototype().GetObject("unix")
			unix = val.Value().(int64)
		}

		return NewInstance(t.Add(-time.Second * time.Duration(unix)))
	}))

	proto.SetObject("addDays", n.Function(n.Describe(n.Arg("days", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		days := int(a.Name("days").Value().(int64))
		return NewInstance(t.AddDate(0, 0, days))
	}))

	proto.SetObject("addMonths", n.Function(n.Describe(n.Arg("months", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		months := int(a.Name("months").Value().(int64))
		return NewInstance(t.AddDate(0, months, 0))
	}))

	proto.SetObject("addYears", n.Function(n.Describe(n.Arg("years", n.TInt)).Returns(inst.Type()), func(a *n.Args) (any, error) {
		reCalc()
		years := int(a.Name("years").Value().(int64))
		return NewInstance(t.AddDate(years, 0, 0))
	}))

	proto.SetObject("year", intFn(func() int {
		reCalc()
		return t.Year()
	}))
	proto.SetObject("month", intFn(func() int {
		reCalc()
		return int(t.Month())
	}))
	proto.SetObject("day", intFn(func() int {
		reCalc()
		return t.Day()
	}))
	proto.SetObject("hour", intFn(func() int {
		reCalc()
		return t.Hour()
	}))
	proto.SetObject("minute", intFn(func() int {
		reCalc()
		return t.Minute()
	}))
	proto.SetObject("second", intFn(func() int {
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
