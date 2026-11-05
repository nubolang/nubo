package interpreter

import (
	"context"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/source"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/packer"
)

type Runtime interface {
	Source() *source.Loader
	GetBuiltin(name string) (language.Object, bool)
	GetEventProvider() events.Provider
	NewID() uint
	RemoveInterpreter(id uint)
	ImportPackage(ctx context.Context, name string, dg *debug.Debug) (language.Object, bool, error)
	GetPacker() (*packer.Packer, error)
	FindInterpreter(file string) (*Interpreter, bool)
	AddInterpreter(file string, interpreter *Interpreter)
	SetInterpreterReturn(id uint, value language.Object)
	GetInterpreterReturn(id uint) (language.Object, bool)
}
