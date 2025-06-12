package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

type Iterator interface {
	// Iterator is an interface that returns a Next function which
	// returns a key, value, and ok (if ok is false, both key and value are nil and the cycle ends)
	Iterator() func() (language.Object, language.Object, bool)
}

func (i *Interpreter) handleFor(node *astnode.Node) (language.Object, error) {
	kv, ok := node.Value.(*astnode.ForValue)
	if !ok {
		return nil, newErr(ErrValueError, fmt.Sprintf("expected a valid for cycle"), node.Debug)
	}

	expr, err := i.evaluateExpression(node.Args[0])
	if err != nil {
		return nil, err
	}

	iterator, ok := expr.(Iterator)
	if !ok {
		return nil, newErr(ErrValueError, fmt.Sprintf("expected interator, got %s: %s", expr.Type(), expr.Value()), expr.Debug())
	}

	iterate := iterator.Iterator()

	for {
		key, value, ok := iterate()
		if !ok {
			break
		}

		ir := NewWithParent(i, ScopeBlock)
		if kv.Iterator != nil {
			err := ir.Declare(kv.Iterator.Value.(string), key, key.Type(), true)
			if err != nil {
				return nil, err
			}
		}
		if kv.Value != nil {
			err := ir.Declare(kv.Value.Value.(string), value, value.Type(), true)
			if err != nil {
				return nil, err
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, err
		}
		if ob != nil {
			return ob, nil
		}
	}

	return nil, nil
}
