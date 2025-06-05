package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type Dict struct {
	Data      map[Object]Object
	KeyType   ObjectComplexType
	ValueType ObjectComplexType
	debug     *debug.Debug
}

func NewDict(keys []Object, values []Object, keyType ObjectComplexType, valueType ObjectComplexType, debug *debug.Debug) (*Dict, error) {
	var m = make(map[Object]Object, len(keys))
	for i, key := range keys {
		if !key.Type().Base().Hashable() {
			return nil, fmt.Errorf("key %s is not hashable", key.Inspect())
		}

		m[key] = values[i]
	}

	return &Dict{
		Data:      m,
		KeyType:   keyType,
		ValueType: valueType,
		debug:     debug,
	}, nil
}

func (i *Dict) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Dict) Type() ObjectComplexType {
	return TypeDict
}

func (i *Dict) Inspect() string {
	return fmt.Sprintf("<Object(dict[%s, %s] @ %s)>", i.KeyType.String(), i.ValueType.String(), i.String())
}

func (i *Dict) TypeString() string {
	return fmt.Sprintf("<Object(dict[%s, %s])>", i.KeyType.String(), i.ValueType.String())
}

func (i *Dict) String() string {
	var itemsString []string
	for key, value := range i.Data {
		itemsString = append(itemsString, fmt.Sprintf("%v: %s", key, value.String()))
	}

	return fmt.Sprintf("dict[%s]", strings.Join(itemsString, ", "))
}

func (i *Dict) GetPrototype() Prototype {
	return nil
}

func (i *Dict) Value() any {
	return i.Data
}

func (i *Dict) Debug() *debug.Debug {
	return i.debug
}

func (i *Dict) Clone() Object {
	var m = make(map[Object]Object, len(i.Data))
	for key, value := range i.Data {
		m[key] = value.Clone()
	}

	return &Dict{
		Data:      m,
		KeyType:   i.KeyType,
		ValueType: i.ValueType,
		debug:     i.debug,
	}
}
