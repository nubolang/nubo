package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleWhile(node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, runExc("expected a valid while statement").WithDebug(node.Debug)
	}

	condition := func() (bool, error) {
		ok, err := i.eval(node.Args[0])
		if err != nil {
			return false, wrapRunExc(err, node.Debug)
		}
		if ok.Type() != language.TypeBool {
			return false, valueExc("expected bool, got %s with value %s", ok.Type(), ok.Value()).WithDebug(ok.Debug())
		}
		return ok.Value().(bool), nil
	}

	for {
		ok, err := condition()
		if err != nil {
			return nil, wrapRunExc(err, node.Debug)
		}

		if !ok {
			break
		}

		ir := NewWithParent(i, ScopeBlock, "while")
		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, wrapRunExc(err, node.Debug)
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				if ob.String() == "break" {
					break
				}
				if ob.String() == "continue" {
					continue
				}
				return nil, runExc("invalid language signal: %q", ob.String()).WithDebug(ob.Debug())
			}
			return ob, nil
		}
	}

	return nil, nil
}
