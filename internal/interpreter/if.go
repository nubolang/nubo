package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleIf(node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, newErr(ErrAst, "Something went wrong with creating if statement", node.Debug)
	}

	condition := func() (bool, error) {
		ok, err := i.evaluateExpression(node.Args[0])
		if err != nil {
			return false, err
		}
		if ok.Type() != language.TypeBool {
			return false, newErr(ErrValueError, fmt.Sprintf("expected bool, got %s: %s", ok.Type(), ok.Value()), ok.Debug())
		}
		return ok.Value().(bool), nil
	}

	ok, err := condition()
	if err != nil {
		return nil, err
	}

	var execNodes []*astnode.Node
	if ok {
		execNodes = node.Body
	} else {
		execNodes = node.Children
	}

	if len(execNodes) > 0 {
		ir := NewWithParent(i, ScopeBlock)
		ob, err := ir.Run(execNodes)
		if err != nil {
			return nil, err
		}
		if ob != nil {
			return ob, nil
		}
	}

	return nil, nil
}
