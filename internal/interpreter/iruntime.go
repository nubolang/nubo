package interpreter

import (
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/packer"
)

type Runtime interface {
	GetBuiltin(name string) (language.Object, bool)
	GetEventProvider() events.Provider
	NewID() uint
	RemoveInterpreter(id uint)
	ImportPackage(name string, dg *debug.Debug) (language.Object, bool)
	GetPacker() (*packer.Packer, error)
	FindInterpreter(file string) (*Interpreter, bool)
	AddInterpreter(file string, interpreter *Interpreter)
}
