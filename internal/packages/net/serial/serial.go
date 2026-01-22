package serial

import (
	"context"
	"time"

	goserial "go.bug.st/serial"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var portStruct *language.Struct

func NewSerial(dg *debug.Debug) language.Object {
	instance := n.NewPackage("serial", dg)
	proto := instance.GetPrototype()

	if portStruct == nil {
		portStruct = language.NewStruct("Port", []language.StructField{}, dg)
		ctx := context.Background()
		proto := portStruct.GetPrototype().(*language.StructPrototype)
		proto.Unlock()
		defer proto.Lock()

		proto.SetObject(ctx, "init", n.Function(
			n.Describe(
				n.Arg("device", n.TString),
				n.Arg("baud", n.TInt, language.NewInt(9600, dg)),
			).Returns(portStruct.Type()),
			openPort,
		))
	}

	ctx := context.Background()

	proto.SetObject(ctx, "Port", portStruct)

	return instance
}

func openPort(args *n.Args) (any, error) {
	device := args.Name("device").String()
	baud := int(args.Name("baud").Value().(int64))

	mode := &goserial.Mode{BaudRate: baud}
	p, err := goserial.Open(device, mode)
	if err != nil {
		return nil, err
	}

	inst, err := portStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	proto := inst.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	proto.SetObject(ctx, "write", n.Function(
		n.Describe(n.Arg("data", n.TString)),
		func(a *n.Args) (any, error) {
			_, err := p.Write([]byte(a.Name("data").String()))
			return nil, err
		},
	))

	proto.SetObject(ctx, "read", n.Function(
		n.Describe(
			n.Arg("timeoutMs", n.TInt),
		).Returns(n.TString),
		func(a *n.Args) (any, error) {
			tm := a.Name("timeoutMs")
			timeout := time.Duration(tm.Value().(int64)) * time.Millisecond
			buf := make([]byte, 4096)

			_ = p.SetReadTimeout(timeout)
			nr, err := p.Read(buf)
			if err != nil {
				return nil, err
			}

			return n.String(string(buf[:nr]), tm.Debug()), nil
		},
	))

	proto.SetObject(ctx, "close", n.Function(
		n.Describe(),
		func(a *n.Args) (any, error) {
			return nil, p.Close()
		},
	))

	return inst, nil
}
