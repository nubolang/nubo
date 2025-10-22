package language

import (
	"context"
	"fmt"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
	"go.uber.org/zap"
)

type StructInstance struct {
	base      *Struct
	Name      string
	prototype *StructPrototype
	bucket    map[string]any
	debug     *debug.Debug
}

func NewStructInstance(base *Struct, name string, debug *debug.Debug) (*StructInstance, error) {
	s := &StructInstance{
		base:  base,
		Name:  name,
		debug: debug,
	}

	proto, err := base.prototype.NewInstance(s)
	if err != nil {
		return nil, err
	}

	s.prototype = proto

	return s, nil
}

func (i *StructInstance) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *StructInstance) Type() *Type {
	cloned := i.base.structType.DeepClone()
	cloned.BaseType = ObjectTypeStructInstance
	return cloned
}

func (i *StructInstance) Inspect() string {
	objs := i.GetPrototype().Objects()
	if len(objs) == 0 {
		return fmt.Sprintf("(struct %s) %s{}", i.Type().ID, i.Name)
	}

	var items []string = make([]string, 0, len(objs))
	for name, item := range objs {
		if _, ok := i.base.privateMap[name]; ok {
			name = fmt.Sprintf("(private) %s", name)
		}
		items = append(items, fmt.Sprintf("%s: %s", name, indentString(item.Type().String(), "\t")))
	}

	return fmt.Sprintf(
		"(struct %s) %s{\n\t%s\n}",
		i.Type().ID,
		i.Name,
		strings.Join(items, ",\n\t"),
	)
}

func (i *StructInstance) TypeString() string {
	return "<Object(struct)>"
}

func (i *StructInstance) String() string {
	if proto := i.GetPrototype(); proto != nil {
		if str, ok := proto.GetObject(context.Background(), "__string__"); ok {
			if strFn, ok := str.(*Function); ok {
				if len(strFn.ArgTypes) == 0 && strFn.ReturnType.Compare(TypeString) {
					if s, err := strFn.Data(context.Background(), nil); err == nil {
						return s.String()
					}
				}
			}
		}
	}

	if len(i.base.Data) == 0 {
		return fmt.Sprintf("(struct) %s{}", i.Name)
	}

	var items = make([]string, len(i.base.Data))
	for inx, field := range i.base.Data {
		ob, ok := i.GetPrototype().GetObject(context.Background(), field.Name)
		if !ok {
			items[inx] = fmt.Sprintf("%s: <invalid>", field.Name)
		} else {
			items[inx] = fmt.Sprintf("%s: %s", field.Name, indentString(ob.String(), "\t"))
		}
	}

	return fmt.Sprintf(
		"(struct) %s{\n\t%s\n}",
		i.Name,
		strings.Join(items, ",\n\t"),
	)
}

func (i *StructInstance) GetPrototype() Prototype {
	return i.prototype
}

func (i *StructInstance) Value() any {
	return i
}

func (i *StructInstance) Debug() *debug.Debug {
	return i.debug
}

func (i *StructInstance) Clone() Object {
	proto := i.GetPrototype()
	if proto == nil {
		return i
	}

	cl, ok := proto.GetObject(context.Background(), "__clone__")
	if ok {
		clFn, ok := cl.(*Function)
		if !ok {
			zap.L().Fatal(i.String(), zap.String("warn", " __clone__ should be a function"))
			return i
		}
		cl, err := clFn.Data(context.Background(), nil)
		if err != nil {
			zap.L().Fatal(i.String(), zap.String("error", err.Error()))
			return i
		}
		return cl
	}

	return i
}

func (s *StructInstance) Iterator() func() (Object, Object, bool) {
	var (
		inx       = 0
		fieldsLen = len(s.base.Data)
	)

	return func() (Object, Object, bool) {
		if inx >= fieldsLen {
			return nil, nil, false
		}

		key := s.base.Data[inx]
		value, ok := s.GetPrototype().GetObject(context.Background(), key.Name)
		if !ok {
			zap.L().Warn(s.String(), zap.String("field not found", key.Name))
			return nil, nil, false
		}

		inx++
		return NewString(key.Name, s.debug), value, true
	}
}

func (s *StructInstance) BucketGet(key string) (any, bool) {
	if s.bucket == nil {
		return nil, false
	}
	return s.bucket[key], true
}

func (s *StructInstance) BucketSet(key string, value any) {
	if s.bucket == nil {
		s.bucket = make(map[string]any)
	}
	s.bucket[key] = value
}
