package language

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
)

type String struct {
	Data      string
	prototype *StringPrototype
	debug     *debug.Debug
}

func NewString(value string, debug *debug.Debug) *String {
	return &String{
		Data:      value,
		prototype: nil,
		debug:     debug,
	}
}

func (i *String) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *String) Type() *Type {
	return TypeString
}

func (i *String) Inspect() string {
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("(string) %s {}", strconv.Quote(i.String()))
	}

	var items []string = make([]string, 0, len(objs))
	for name, item := range objs {
		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.Type().String(), "\t")))
	}

	return fmt.Sprintf(
		"(string) %s {\n\t%s\n}",
		strconv.Quote(i.String()),
		strings.Join(items, ",\n\t"),
	)
}

func (i *String) TypeString() string {
	return "<Object(string)>"
}

func (i *String) String() string {
	return i.Data
}

func (i *String) GetPrototype() Prototype {
	if i.prototype == nil {
		i.prototype = NewStringPrototype(i)
	}
	return i.prototype
}

func (i *String) Value() any {
	return i.Data
}

func (i *String) Debug() *debug.Debug {
	return i.debug
}

func (i *String) Clone() Object {
	return NewString(i.Data, i.debug)
}

func (s *String) Iterator() func() (Object, Object, bool) {
	var (
		parts    = []rune(s.Data)
		inx      = 0
		partsLen = len(parts)
	)

	return func() (Object, Object, bool) {
		if inx >= partsLen {
			return nil, nil, false
		}

		value := parts[inx]
		key := NewInt(int64(inx), s.debug)

		inx++
		return key, NewChar(value, s.debug), true
	}
}
