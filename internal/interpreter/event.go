package interpreter

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"go.uber.org/zap"
)

func (i *Interpreter) handleEventDecl(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.event.declare.start", zap.Uint("id", i.ID), zap.String("event", node.Content))

	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		zap.L().Error("interpreter.event.declare.lookup", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Error(err))
		return nil, err
	}

	eventProvider := i.runtime.GetEventProvider()
	event := &events.Event{
		ID:   fmt.Sprintf("%d_%s", iid, name),
		Args: make([]language.FnArg, len(node.Args)),
	}

	for j, child := range node.Args {
		typ, err := i.parseTypeNode(child.ValueType)
		if err != nil {
			return nil, err
		}

		event.Args[j] = &language.BasicFnArg{
			TypeVal: typ,
			NameVal: child.Content,
		}
	}

	eventProvider.AddEvent(event)
	zap.L().Debug("interpreter.event.declare.success", zap.Uint("id", i.ID), zap.String("event", node.Content))

	return nil, nil
}

func (i *Interpreter) handleSubscribe(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.event.subscribe.start", zap.Uint("id", i.ID), zap.String("event", node.Content))

	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		zap.L().Error("interpreter.event.subscribe.lookup", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Error(err))
		return nil, err
	}

	eventProvider := i.runtime.GetEventProvider()

	unsub, err := eventProvider.Subscribe(fmt.Sprintf("%d_%s", iid, name), func(td events.TransportData) {
		ir := NewWithParent(i, ScopeBlock)

		for i, arg := range node.Args {
			if len(arg.Body) != 1 {
				return
			}

			id := arg.Body[0].Value.(string)
			if arg.Body[0].Kind != "IDENTIFIER" || strings.Contains(id, ".") {
				return
			}

			ir.Declare(id, td[i], td[i].Type(), true)
		}

		_, _ = ir.Run(node.Body)
	})

	i.unsub = append(i.unsub, unsub)
	zap.L().Debug("interpreter.event.subscribe.success", zap.Uint("id", i.ID), zap.String("event", node.Content))

	return nil, err
}

func (i *Interpreter) handlePublish(node *astnode.Node) (language.Object, error) {
	zap.L().Debug("interpreter.event.publish.start", zap.Uint("id", i.ID), zap.String("event", node.Content))

	name, iid, err := i.getEventByName(node.Content, node.Debug)
	if err != nil {
		zap.L().Error("interpreter.event.publish.lookup", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Error(err))
		return nil, err
	}
	eventID := fmt.Sprintf("%d_%s", iid, name)

	eventProvider := i.runtime.GetEventProvider()
	event := eventProvider.GetEvent(eventID)

	if len(event.Args) != len(node.Args) {
		err := argError(len(event.Args), len(node.Args)).WithDebug(node.Debug)
		zap.L().Error("interpreter.event.publish.argCount", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Error(err))
		return nil, err
	}

	args := make(events.TransportData, len(node.Args))
	for j, arg := range node.Args {
		value, err := i.eval(arg)
		if err != nil {
			zap.L().Error("interpreter.event.publish.argEval", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Int("index", j), zap.Error(err))
			return nil, err
		}

		if !language.TypeCheck(event.Args[j].Type(), value.Type()) {
			err := typeMismatch(event.Args[j].Type(), value.Type()).WithDebug(node.Debug)
			zap.L().Error("interpreter.event.publish.typeMismatch", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Int("index", j), zap.Error(err))
			return nil, err
		}
		args[j] = value.Clone()
	}

	err = eventProvider.Publish(eventID, args)
	if err != nil {
		zap.L().Error("interpreter.event.publish.error", zap.Uint("id", i.ID), zap.String("event", node.Content), zap.Error(err))
		return nil, err
	}

	zap.L().Debug("interpreter.event.publish.success", zap.Uint("id", i.ID), zap.String("event", node.Content))

	return nil, err
}

func (i *Interpreter) getEventByName(name string, d *debug.Debug) (string, uint, error) {
	zap.L().Debug("interpreter.event.lookup.start", zap.Uint("id", i.ID), zap.String("event", name))

	var iid = i.ID

	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			err := runExc("invalid event '%s'", name).WithDebug(d)
			zap.L().Error("interpreter.event.lookup.invalid", zap.Uint("id", i.ID), zap.String("event", name), zap.Error(err))
			return "", 0, err
		}
		imported := parts[0]
		name = parts[1]

		i.mu.RLock()
		ir, ok := i.imports[imported]
		if !ok {
			i.mu.RUnlock()
			err := importError("not found '%s'", imported).WithDebug(d)
			zap.L().Error("interpreter.event.lookup.importMissing", zap.Uint("id", i.ID), zap.String("import", imported), zap.Error(err))
			return "", 0, err
		}
		iid = ir.ID
		i.mu.RUnlock()
	}

	zap.L().Debug("interpreter.event.lookup.success", zap.Uint("id", i.ID), zap.String("event", name), zap.Uint("owner", iid))
	return name, iid, nil
}
