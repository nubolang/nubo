package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

var Nil *NilObj

type NilObj struct{}

func NewNil() *NilObj {
	if Nil == nil {
		Nil = &NilObj{}
	}
	return Nil
}

func (n *NilObj) ID() string {
	return fmt.Sprintf("%p", n)
}

func (n *NilObj) Type() ObjectComplexType {
	return TypeNil
}

func (n *NilObj) Inspect() string {
	return fmt.Sprintf("<Object(nil)>")
}

func (n *NilObj) TypeString() string {
	return "<Object(nil)>"
}

func (n *NilObj) String() string {
	return "nil"
}

func (n *NilObj) GetPrototype() Prototype {
	return nil
}

func (n *NilObj) Value() any {
	return nil
}

func (n *NilObj) Debug() *debug.Debug {
	return nil
}

func (n *NilObj) Clone() Object {
	return n
}
