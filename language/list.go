package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type List struct {
	Data     []Object
	ItemType *Type
	proto    *ListPrototype
	debug    *debug.Debug
}

func NewList(values []Object, itemType *Type, debug *debug.Debug) *List {
	return &List{
		Data:     values,
		ItemType: itemType,
		debug:    debug,
	}
}

func (i *List) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *List) Type() *Type {
	return &Type{
		BaseType: ObjectTypeList,
		Element:  i.ItemType,
	}
}

func (i *List) Inspect() string {
	return fmt.Sprintf("<Object(list[%s] @ %s)>", i.ItemType.String(), i.String())
}

func (i *List) TypeString() string {
	return fmt.Sprintf("<Object(list[%s])>", i.ItemType.String())
}

func (i *List) String() string {
	var itemsString []string
	for _, item := range i.Data {
		itemsString = append(itemsString, item.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(itemsString, ", "))
}

func (i *List) GetPrototype() Prototype {
	if i.proto == nil {
		i.proto = NewListPrototype(i)
	}
	return i.proto
}

func (i *List) Value() any {
	return i.Data
}

func (i *List) Debug() *debug.Debug {
	return i.debug
}

func (i *List) Clone() Object {
	return NewList(i.Data, i.ItemType, i.debug)
}

func (i *List) Iterator() func() (Object, Object, bool) {
	var inx = 0

	return func() (Object, Object, bool) {
		if inx >= len(i.Data) {
			return nil, nil, false
		}

		value := i.Data[inx]
		key := NewInt(int64(inx), i.debug)

		inx++
		return key, value, true
	}
}
