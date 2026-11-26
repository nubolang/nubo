package thread

import (
	"context"
	"fmt"
	"sync"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var portalOnce sync.Once

var portalStruct *language.Struct

func newPortalStruct(dg *debug.Debug) {
	portalOnce.Do(func() {

		ctx := context.Background()

		portalStruct = language.NewStruct("Portal", []language.StructField{
			{
				Name:    "capacity",
				Type:    n.TInt,
				Private: true,
			},
			{
				Name:    "closed",
				Type:    n.TBool,
				Private: true,
			},
		}, dg)

		proto := portalStruct.GetPrototype().(*language.StructPrototype)
		proto.Unlock()

		proto.SetObject(ctx, "init", n.Function(n.Describe(
			n.Arg("self", portalStruct.Type()),
			n.Arg("capacity", n.TInt, n.Int(16, dg)),
		).Returns(portalStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				capacity := a.Name("capacity")
				if capacity.Value().(int64) < 0 {
					return nil, fmt.Errorf("capacity must be nonnegative")
				}
				proto := self.GetPrototype()

				proto.SetObject(language.StructSetPrivate(ctx), "capacity", capacity)
				proto.SetObject(language.StructSetPrivate(ctx), "closed", n.Bool(false, dg))

				bucket := self.(language.Bucketable)
				bucket.BucketSet("portal", NewPortal(int(capacity.Value().(int64))))

				return self, nil
			}))

		proto.SetObject(ctx, "close", n.Function(n.Describe(
			n.Arg("self", portalStruct.Type()),
		),
			func(a *n.Args) (any, error) {
				self := a.Name("self")

				bucket := self.(language.Bucketable)
				p, _ := bucket.BucketGet("portal")
				p.(*Portal).Close()

				return nil, nil
			}))

		getPortal := func(self language.Object) *Portal {
			p, _ := self.(language.Bucketable).BucketGet("portal")
			return p.(*Portal)
		}
		isClosed := func(self language.Object) error {
			if getPortal(self).IsClosed() {
				return fmt.Errorf("[thread/Portal] portal is closed")
			}
			return nil
		}

		proto.SetObject(ctx, "send", n.Function(n.Describe(
			n.Arg("self", portalStruct.Type()),
			n.Arg("message", n.TAny),
		),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				if err := isClosed(self); err != nil {
					return nil, err
				}

				message := a.Name("message")

				bucket := self.(language.Bucketable)
				p, _ := bucket.BucketGet("portal")
				return nil, p.(*Portal).Send(message)
			}))

		proto.SetObject(ctx, "receive", n.Function(n.Describe(
			n.Arg("self", portalStruct.Type()),
			n.Arg("timeout", n.Nullable(n.TInt), language.Nil),
		).Returns(n.TAny),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				if err := isClosed(self); err != nil {
					return nil, err
				}

				bucket := self.(language.Bucketable)
				p, _ := bucket.BucketGet("portal")
				po := p.(*Portal)

				timeout := a.Name("timeout").Value()
				if timeout == nil {
					return po.ReceiveMessage()
				}

				return po.ReceiveWithTimeout(int(timeout.(int64)))
			}))

		proto.SetObject(ctx, "trySend", n.Function(n.Describe(n.Arg("self", portalStruct.Type()), n.Arg("message", n.TAny)).Returns(n.TBool), func(a *n.Args) (any, error) { return getPortal(a.Name("self")).TrySend(a.Name("message")), nil }))
		proto.SetObject(ctx, "tryReceive", n.Function(n.Describe(n.Arg("self", portalStruct.Type())).Returns(n.TAny), func(a *n.Args) (any, error) {
			value, ok, closed := getPortal(a.Name("self")).TryReceive()
			if value == nil {
				value = language.Nil
			}
			return map[string]any{"value": value, "ok": ok, "closed": closed}, nil
		}))
		proto.SetObject(ctx, "length", n.Function(n.Describe(n.Arg("self", portalStruct.Type())).Returns(n.TInt), func(a *n.Args) (any, error) { return getPortal(a.Name("self")).Length(), nil }))
		proto.SetObject(ctx, "isClosed", n.Function(n.Describe(n.Arg("self", portalStruct.Type())).Returns(n.TBool), func(a *n.Args) (any, error) { return getPortal(a.Name("self")).IsClosed(), nil }))
		proto.SetObject(ctx, "drain", n.Function(n.Describe(n.Arg("self", portalStruct.Type())).Returns(n.TAny), func(a *n.Args) (any, error) {
			return language.NewList(getPortal(a.Name("self")).Drain(), n.TAny, nil), nil
		}))
		proto.Lock()
		proto.Implement()
	})
}
