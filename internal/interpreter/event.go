package interpreter

import (
	"fmt"
	"strings"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/internal/pubsub"
	"github.com/nubogo/nubo/language"
)

func (i *Interpreter) handleEventDecl(node *astnode.Node) (language.Object, error) {
	name := node.Content
	eventProvider := i.runtime.GetEventProvider()
	event := pubsub.Event{
		ID:   fmt.Sprintf("%s_%s", i.ID, name),
		Args: make([]language.FnArg, len(node.Children)),
	}

	for j, child := range node.Children {
		typ, err := i.stringToType(child.ValueType.Content)
		if err != nil {
			return nil, err
		}

		event.Args[j] = &language.BasicFnArg{
			TypeVal: typ,
			NameVal: child.Content,
		}
	}

	eventProvider.AddEvent(event)

	return nil, nil
}

func (i *Interpreter) handleSubscribe(node *astnode.Node) (language.Object, error) {
	name := node.Content
	eventProvider := i.runtime.GetEventProvider()

	unsub, err := eventProvider.Subscribe(fmt.Sprintf("%s_%s", i.ID, name), func(td pubsub.TransportData) {
		ir := NewWithParent(i, ScopeFunction)

		for i, arg := range node.Args {
			if len(arg.Body) != 1 {
				return
			}

			id := arg.Body[0].Value.(string)
			if arg.Body[0].Kind != "IDENTIFIER" || strings.Contains(id, ".") {
				return
			}

			ir.BindObject(id, td[i], true)
		}

		_, _ = ir.Run(node.Body)
	})

	i.unsub = append(i.unsub, unsub)

	return nil, err
}

func (i *Interpreter) handlePublish(node *astnode.Node) (language.Object, error) {
	name := node.Content
	eventProvider := i.runtime.GetEventProvider()

	args := make(pubsub.TransportData, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.fromExpression(arg)
		if err != nil {
			return nil, err
		}
		args[j] = value
	}

	err := eventProvider.Publish(fmt.Sprintf("%s_%s", i.ID, name), args)

	return nil, err
}
