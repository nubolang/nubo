package language

import (
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type Dict struct {
	Data      *OrderedMap
	KeyType   *Type
	ValueType *Type
	proto     *DictPrototype
	debug     *debug.Debug
}

func NewDict(keys []Object, values []Object, keyType *Type, valueType *Type, debug *debug.Debug) (*Dict, error) {
	m := NewOrderedMap()
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

		m.Set(key, values[i])
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
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("(dict[%s, %s]) {}", i.KeyType.String(), i.ValueType.String())
	}

	var items []string = make([]string, 0, len(objs))
	for name, item := range objs {
		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.Type().String(), "\t")))
	}

	return fmt.Sprintf(
		"(dict[%s, %s]) {\n\t%s\n}",
		i.KeyType.String(), i.ValueType.String(),
		strings.Join(items, ",\n\t"),
	)
}

func (i *Dict) TypeString() string {
	return fmt.Sprintf("<Object(dict[%s, %s])>", i.KeyType.String(), i.ValueType.String())
}

func (i *Dict) String() string {
	if i.Data.Len() == 0 {
		return "dict{}"
	}

	var items []string
	i.Data.Iterate(func(key Object, value Object) bool {
		items = append(items, fmt.Sprintf("%v: %s", key, indentString(value.String(), "\t")))
		return true
	})

	return fmt.Sprintf(
		"dict{\n\t%s\n}",
		strings.Join(items, ",\n\t"),
	)
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
	m := NewOrderedMap()

	i.Data.Iterate(func(key Object, value Object) bool {
		m.Set(key, value.Clone())
		return true
	})

	return &Dict{
		Data:      m,
		KeyType:   i.KeyType,
		ValueType: i.ValueType,
		debug:     i.debug,
	}
}

func (i *Dict) Iterator() func() (Object, Object, bool) {
	var current *orderedEntry = i.Data.head

	return func() (Object, Object, bool) {
		if current == nil {
			return nil, nil, false
		}

		key := current.key
		value := current.value
		current = current.next
		return key, value, true
	}
}
