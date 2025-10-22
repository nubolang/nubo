package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleImpl(node *astnode.Node) error {
	name := node.Content
	definition, ok := i.GetObject(name)
	if !ok || definition.Type().Base() != language.ObjectTypeStructDefinition {
		return runExc("cannot implement object %q", name).WithDebug(node.Debug)
	}

	proto, ok := definition.GetPrototype().(*language.StructPrototype)
	if !ok {
		return runExc("cannot implement object %q, no prototype found", name).WithDebug(node.Debug)
	}

	if proto.Implemented() {
		return runExc("cannot re-implement %q, already implemented", name).WithDebug(node.Debug)
	}

	proto.Unlock()

	for _, child := range node.Body {
		name := child.Content
		fn, err := i.handleFunctionDecl(child, true)
		if err != nil {
			return wrapRunExc(err, child.Debug)
		}

		ctx := i.ctx
		if child.Flags.Contains("PRIVATE") {
			if name == "init" {
				return runExc("struct (%s) \"init\" hook method cannot be private", node.Content).WithDebug(child.Debug)
			}
			ctx = language.StructSetPrivate(ctx)
		}

		if err := proto.SetObject(ctx, name, fn); err != nil {
			return wrapRunExc(err, node.Debug)
		}
	}

	proto.Lock()
	proto.Implement()

	return nil
}
