package language

import (
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/debug"
)

var TypeNumber = NewUnionType(TypeInt, TypeFloat)

type Int struct {
	Data  int64
	proto *IntPrototype
	debug *debug.Debug
}

func NewInt(value int64, debug *debug.Debug) *Int {
	return &Int{
		Data:  value,
		debug: debug,
	}
}

func (i *Int) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Int) Type() *Type {
	return TypeInt
}

func (i *Int) Inspect() string {
	return fmt.Sprintf("<Object(int @ %s)>", i.String())
}

func (i *Int) TypeString() string {
	return "<Object(int)>"
}

func (i *Int) String() string {
	return strconv.Itoa(int(i.Data))
}

func (i *Int) GetPrototype() Prototype {
	if i.proto == nil {
		i.proto = NewIntPrototype(i)
	}
	return i.proto
}

func (i *Int) Value() any {
	return i.Data
}

func (i *Int) Debug() *debug.Debug {
	return i.debug
}

func (i *Int) Clone() Object {
	return NewInt(i.Data, i.debug)
}

func (i *Int) Iterator() func() (Object, Object, bool) {
	var inx int64 = 0

	return func() (Object, Object, bool) {
		if inx >= i.Data {
			return nil, nil, false
		}

		key := NewInt(inx, i.debug)

		inx++
		return key, key, true
	}
}
