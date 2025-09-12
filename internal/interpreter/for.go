package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

type Iterator interface {
	// Iterator is an interface that returns a Next function which
	// returns a key, value, and ok (if ok is false, both key and value are nil and the cycle ends)
	Iterator() func() (language.Object, language.Object, bool)
}

func (i *Interpreter) handleFor(node *astnode.Node) (language.Object, error) {
	kv, ok := node.Value.(*astnode.ForValue)
	if !ok {
		return nil, newErr(ErrValueError, "expected a valid for cycle", node.Debug)
	}

	expr, err := i.eval(node.Args[0])
	if err != nil {
		return nil, err
	}

	iterator, ok := expr.(Iterator)
	if !ok {
		return nil, newErr(ErrValueError, fmt.Sprintf("expected iterator, got %s(%v)", expr.Type(), expr.Value()), expr.Debug())
	}

	iterate := iterator.Iterator()

	// Create loop scope only once
	ir := NewWithParent(i, ScopeBlock, "for")

	var keyName, valName string
	if kv.Iterator != nil {
		keyName = kv.Iterator.Value.(string)
		// declare once
		if err := ir.Declare(keyName, language.Nil, n.TAny, true); err != nil {
			return nil, err
		}
	}
	if kv.Value != nil {
		valName = kv.Value.Value.(string)
		// declare once
		if err := ir.Declare(valName, language.Nil, n.TAny, true); err != nil {
			return nil, err
		}
	}

	for {
		key, value, ok := iterate()
		if !ok {
			break
		}

		// only assign instead of redeclare
		if keyName != "" {
			if err := ir.Assign(keyName, key); err != nil {
				return nil, err
			}
		}
		if valName != "" {
			if err := ir.Assign(valName, value); err != nil {
				return nil, err
			}
		}

		ob, err := ir.Run(node.Body)
		if err != nil {
			return nil, err
		}
		if ob != nil {
			if ob.Type().Base() == language.ObjectTypeSignal {
				switch ob.String() {
				case "break":
					break
				case "continue":
					continue
				default:
					return nil, newErr(ErrInvalid,
						fmt.Sprintf("invalid language signal: %s", ob.String()), ob.Debug())
				}
			}
			return ob, nil
		}
	}

	return nil, nil
}
