package interpreter

import (
	"sync"

	"github.com/nubogo/nubo/internal/ast/astnode"
	"github.com/nubogo/nubo/language"
)

type Interpreter struct {
	currentFile string

	mu      sync.RWMutex
	objects map[uint32]*entry
}

func New(currentFile string) *Interpreter {
	return &Interpreter{
		currentFile: currentFile,
		objects:     make(map[uint32]*entry),
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
