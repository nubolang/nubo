package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type Dict struct {
	Data      map[Object]Object
	KeyType   *Type
	ValueType *Type
	proto     *DictPrototype
	debug     *debug.Debug
}

func NewDict(keys []Object, values []Object, keyType *Type, valueType *Type, debug *debug.Debug) (*Dict, error) {
	var m = make(map[Object]Object, len(keys))
	for i, key := range keys {
		if !key.Type().Base().Hashable() {
			return nil, fmt.Errorf("key %s is not hashable", key.Inspect())
		}

		if !keyType.Compare(key.Type()) {
			return nil, fmt.Errorf("key type %s does not match expected type %s", key.Type().String(), keyType.String())
		}

		if !valueType.Compare(values[i].Type()) {
			return nil, fmt.Errorf("value type %s does not match expected type %s", values[i].Type().String(), valueType.String())
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

func (i *Dict) Type() *Type {
	return &Type{
		BaseType: ObjectTypeDict,
		Key:      i.KeyType,
		Value:    i.ValueType,
	}
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
	if i.proto == nil {
		i.proto = NewDictPrototype(i)
	}
	return i.proto
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
