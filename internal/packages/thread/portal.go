package thread

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var portalStruct *language.Struct

func newPortalStruct(dg *debug.Debug) {
	if portalStruct != nil {
		return
	}

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
			proto := self.GetPrototype()

			proto.SetObject(language.StructSetPrivate(ctx), "closed", n.Bool(true, dg))

			bucket := self.(language.Bucketable)
			p, _ := bucket.BucketGet("portal")
			p.(*Portal).Close()

			return nil, nil
		}))

	isClosed := func(self language.Object) error {
		proto := self.GetPrototype()
		closed, ok := proto.GetObject(language.StructAllowPrivateCtx(ctx), "closed")
		if !ok {
			return nil
		}

		if closed.Value().(bool) {
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
			p.(*Portal).Send(message)

			return nil, nil
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
				return po.Receive(), nil
			}

			return po.ReceiveWithTimeout(int(timeout.(int64)))
		}))

	proto.Lock()
	proto.Implement()
}
