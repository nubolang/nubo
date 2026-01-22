package telnet

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var telnetStruct *language.Struct

func NewTelnet(dg *debug.Debug) language.Object {
	instance := n.NewPackage("telnet", dg)
	proto := instance.GetPrototype()

	if telnetStruct == nil {
		telnetStruct = language.NewStruct("Telnet", []language.StructField{}, dg)
		ctx := context.Background()
		proto := telnetStruct.GetPrototype().(*language.StructPrototype)
		proto.Unlock()
		defer proto.Lock()

		proto.SetObject(ctx, "init", n.Function(
			n.Describe(
				n.Arg("host", n.TString),
				n.Arg("port", n.TInt),
			).Returns(telnetStruct.Type()),
			connectTelnet,
		))
	}

	ctx := context.Background()
	proto.SetObject(ctx, "Telnet", telnetStruct)

	return instance
}

func connectTelnet(args *n.Args) (any, error) {
	host := args.Name("host").String()
	port := args.Name("port").Value().(int64)
	address := net.JoinHostPort(host, strconv.Itoa(int(port)))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	inst, err := telnetStruct.NewInstance()
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
			_, err := conn.Write([]byte(a.Name("data").String()))
			return nil, err
		},
	))

	proto.SetObject(ctx, "read", n.Function(
		n.Describe(n.Arg("timeoutMs", n.TInt)).Returns(n.TString),
		func(a *n.Args) (any, error) {
			timeout := time.Duration(a.Name("timeoutMs").Value().(int64)) * time.Millisecond
			_ = conn.SetReadDeadline(time.Now().Add(timeout))

			reader := bufio.NewReader(conn)
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			return n.String(line, a.Name("timeoutMs").Debug()), nil
		},
	))

	proto.SetObject(ctx, "close", n.Function(
		n.Describe(),
		func(a *n.Args) (any, error) {
			return nil, conn.Close()
		},
	))

	return inst, nil
}
