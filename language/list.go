package language

import (
	"fmt"
	"strings"

	"github.com/nubogo/nubo/internal/debug"
)

type List struct {
	Data     []Object
	ItemType ObjectType
	debug    *debug.Debug
}

func NewList(values []Object, itemType ObjectType, debug *debug.Debug) *List {
	return &List{
		Data:     values,
		ItemType: itemType,
		debug:    debug,
	}
}

func (i *List) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *List) Type() ObjectType {
	return TypeList
}

func (i *List) Inspect() string {
	return fmt.Sprintf("<Object(List[%s] @ %s)>", i.ItemType.String(), i.String())
}

func (i *List) TypeString() string {
	return fmt.Sprintf("<Object(List[%s])>", i.ItemType.String())
}

func (i *List) String() string {
	var itemsString []string
	for _, item := range i.Data {
		itemsString = append(itemsString, item.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(itemsString, ", "))
}

func (i *List) GetPrototype() Prototype {
	return nil
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
