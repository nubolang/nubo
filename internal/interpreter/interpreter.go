package interpreter

import (
	"sync"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/language"
)

type Runtime interface {
	GetBuiltin(name string) (language.Object, bool)
}

type Interpreter struct {
	currentFile string

	runtime Runtime
	objects map[uint32]*entry
	imports map[string]*Interpreter

	mu sync.RWMutex
}

func New(currentFile string, runtime Runtime) *Interpreter {
	return &Interpreter{
		currentFile: currentFile,
		runtime:     runtime,
		objects:     make(map[uint32]*entry),
		imports:     make(map[string]*Interpreter),
	}
}

func (i *Interpreter) Run(nodes []*astnode.Node) (language.Object, error) {
	for _, node := range nodes {
		obj, err := i.handleNode(node)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			return obj, nil
		}
	}
	return nil, nil
}
