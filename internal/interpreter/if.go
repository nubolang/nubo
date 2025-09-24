package interpreter

import (
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleIf(node *astnode.Node) (language.Object, error) {
	if len(node.Args) != 1 {
		return nil, exception.Create("invalid or malformed if statement condition").WithDebug(node.Debug).WithLevel(exception.LevelSemantic)
	}

	condition := func() (bool, error) {
		ok, err := i.eval(node.Args[0])
		if err != nil {
			return false, exception.From(err, node.Args[0].Debug, "failed to evaluate condition: @err")
		}
		if ok.Type() != language.TypeBool {
			return false, typeError("condition expected to be type bool, got %s with value: %s", ok.Type(), ok.Value()).WithDebug(ok.Debug())
		}
		return ok.Value().(bool), nil
	}

	ok, err := condition()
	if err != nil {
		return nil, exception.From(err, node.Debug, "failed to evaluate condition: @err")
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
			return nil, exception.From(err, node.Debug, "failed to execute statement body: @err")
		}
		if ob != nil {
			return ob, nil
		}
	}

	return nil, nil
}
