package plugp

import (
	"context"
	"fmt"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"github.com/nubolang/nubo/plug"
)

var (
	plugStruct  *language.Struct
	plugManager *plug.Manager
)

func NewPlug(dg *debug.Debug) language.Object {
	pkg := n.NewPackage("component", nil)
	proto := pkg.GetPrototype()
	ctx := context.Background()

	if plugManager == nil {
		plugManager = plug.GetManager()
	}

	if plugStruct == nil {
		plugStruct = language.NewStruct("Plug", nil, dg)

		ps := plugStruct.GetPrototype().(*language.StructPrototype)

		empty, _ := language.NewDict(nil, nil, n.TString, n.TAny, dg)
		ps.Unlock()
		ps.SetObject(ctx, "send", n.Function(n.Describe(
			n.Arg("self", plugStruct.Type()),
			n.Arg("action", n.TString),
			n.Arg("props", n.NewDictType(n.TString, n.TAny), empty),
		).Returns(n.NewDictType(n.TString, n.TAny)),
			func(a *n.Args) (any, error) {
				self := a.Name("self").Value().(*language.StructInstance)
				rawPlugin, ok := self.BucketGet("_plugin")
				if !ok {
					return nil, fmt.Errorf("plugin cannot be loaded")
				}
				pl, ok := rawPlugin.(*plug.Plugin)
				if !ok {
					return nil, fmt.Errorf("plugin cannot be loaded")
				}

				ctx, cancel := context.WithTimeout(ctx, time.Second*10)
				defer cancel()

				val, err := language.ToValue(a.Name("props"))
				if err != nil {
					return nil, err
				}

				var data map[string]any
				if err := pl.CallInto(ctx, a.Name("action").String(), val, &data); err != nil {
					return nil, err
				}

				return language.FromValue(data, false, self.Debug())
			}))

		ps.Lock()
		ps.Implement()
	}

	proto.SetObject(ctx, "Plug", plugStruct)
	proto.SetObject(ctx, "require", n.Function(n.Describe(
		n.Arg("mode", n.TString, n.String("stdio", dg)),
		n.Arg("path", n.TString, n.String("./backend", dg)),
		n.Arg("addr", n.Nullable(n.TString), language.Nil),
		n.Arg("token", n.Nullable(n.TString), language.Nil),
	).Returns(plugStruct.Type()), func(a *n.Args) (any, error) {
		var opts []plug.PluginOption
		if token := a.Name("token"); token.Type().Compare(n.TString) {
			opts = append(opts, plug.WithToken(token.String()))
		}

		pl, err := plugManager.Load(a.Name("path").String(), opts...)
		if err != nil {
			return nil, err
		}

		inst, err := plugStruct.NewInstance()
		if err != nil {
			return nil, err
		}

		inst.BucketSet("_plugin", pl)
		return inst, nil
	}))

	return pkg
}
