package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleFunctionDecl(node *astnode.Node) (language.Object, error) {
	var args = make([]language.FnArg, len(node.Args))
	var returnType language.ObjectComplexType

	if node.ValueType != nil {
		rt, err := i.stringToType(node.ValueType.Content)
		if err != nil {
			return nil, err
		}
		returnType = rt
	} else {
		returnType = language.TypeVoid
	}

	for j, arg := range node.Args {
		typ, err := i.stringToType(arg.ValueType.Content)
		if err != nil {
			return nil, err
		}
		args[j] = &language.BasicFnArg{
			NameVal: arg.Content,
			TypeVal: typ,
		}
	}

	fn := language.NewTypedFunction(args, returnType, func(o []language.Object) (language.Object, error) {
		ir := NewWithParent(i, ScopeFunction)

		for j, arg := range args {
			providedArg := o[j]
			if !language.TypeCheck(arg.Type(), providedArg.Type()) {
				return nil, newErr(ErrTypeMismatch, fmt.Sprintf("Expected %s but got %s", arg.Type(), providedArg.Type()), providedArg.Debug())
			}

			if err := ir.BindObject(arg.Name(), providedArg, false); err != nil {
				return nil, err
			}
		}

		return ir.Run(node.Body)
	}, node.Debug)

	return nil, i.BindObject(node.Content, fn, false)
}
