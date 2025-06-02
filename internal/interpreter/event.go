package interpreter

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/pubsub"
	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) handleEventDecl(node *astnode.Node) (language.Object, error) {
	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		return nil, err
	}

	eventProvider := i.runtime.GetEventProvider()
	event := &pubsub.Event{
		ID:   fmt.Sprintf("%d_%s", iid, name),
		Args: make([]language.FnArg, len(node.Args)),
	}

	for j, child := range node.Args {
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
	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		return nil, err
	}

	eventProvider := i.runtime.GetEventProvider()

	unsub, err := eventProvider.Subscribe(fmt.Sprintf("%d_%s", iid, name), func(td pubsub.TransportData) {
		ir := NewWithParent(i, ScopeBlock)

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
	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		return nil, err
	}
	eventID := fmt.Sprintf("%d_%s", iid, name)

	eventProvider := i.runtime.GetEventProvider()
	event := eventProvider.GetEvent(eventID)

	if len(event.Args) != len(node.Args) {
		return nil, newErr(ErrTypeMismatch, fmt.Sprintf("argument count mismatch: expected %d, got %d", len(event.Args), len(node.Args)), node.Debug)
	}

	args := make(pubsub.TransportData, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.evaluateExpression(arg)
		if err != nil {
			return nil, err
		}

		if !language.TypeCheck(event.Args[j].Type(), value.Type()) {
			return nil, newErr(ErrTypeMismatch, fmt.Sprintf("type mismatch: expected %s, got %s", event.Args[j].Type(), value.Type()), node.Debug)
		}
		args[j] = value
	}

	err = eventProvider.Publish(eventID, args)

	return nil, err
}

func (i *Interpreter) getEventByName(name string, d *debug.Debug) (string, uint, error) {
	var iid = i.ID

	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			return "", 0, newErr(ErrUnsupported, fmt.Sprintf("invalid event name: %s", name), d)
		}
		imported := parts[0]
		name = parts[1]

		i.mu.RLock()
		ir, ok := i.imports[imported]
		if !ok {
			i.mu.RUnlock()
			return "", 0, newErr(ErrUnsupported, fmt.Sprintf("import not found: %s", imported), d)
		}
		iid = ir.ID
		i.mu.RUnlock()
	}

	return name, iid, nil
}
