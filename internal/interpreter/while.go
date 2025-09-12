package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleWhile(node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, newErr(ErrAst, "Something went wrong with creating while statement", node.Debug)
	}

	condition := func() (bool, error) {
		ok, err := i.eval(node.Args[0])
		if err != nil {
			return false, err
		}
		if ok.Type() != language.TypeBool {
			return false, newErr(ErrValueError, fmt.Sprintf("expected bool, got %s: %s", ok.Type(), ok.Value()), ok.Debug())
		}
		return ok.Value().(bool), nil
	}

	for {
		ok, err := condition()
		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}

		ir := NewWithParent(i, ScopeBlock, "while")
		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, err
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				if ob.String() == "break" {
					break
				}
				if ob.String() == "continue" {
					continue
				}
				return nil, newErr(ErrInvalid, fmt.Sprintf("invalid language singnal: %s", ob.String()), ob.Debug())
			}
			return ob, nil
		}
	}

	return nil, nil
}
