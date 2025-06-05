package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type Ref struct{ Data Object }

func NewRef(value Object) *Ref {
	return &Ref{
		Data: value,
	}
}

func (i *Ref) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Ref) Type() ObjectComplexType {
	return i.Data.Type()
}

func (i *Ref) Inspect() string {
	return fmt.Sprintf("[REF<%s>]", i.Data.Inspect())
}

func (i *Ref) TypeString() string {
	return fmt.Sprintf("[REF<%s>]", i.Data.TypeString())
}

func (i *Ref) String() string {
	return i.Data.String()
}

func (i *Ref) GetPrototype() Prototype {
	return i.Data.GetPrototype()
}

func (i *Ref) Value() any {
	return i.Data.Value()
}

func (i *Ref) Debug() *debug.Debug {
	return i.Data.Debug()
}

func (i *Ref) Clone() Object {
	return i
}
