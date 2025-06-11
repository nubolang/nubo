package n

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

// Bool creates a language.Bool poiter
func Bool(b bool, dg ...*debug.Debug) *language.Bool {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewBool(b, d)
}

// Byte creates a language.Byte poiter
func Byte(b byte, dg ...*debug.Debug) *language.Byte {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewByte(b, d)
}

// Char creates a language.Char poiter
func Char(c rune, dg ...*debug.Debug) *language.Char {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewChar(c, d)
}

// Dict creates a language.Dict poiter
func Dict(dict map[any]any, dg ...*debug.Debug) (*language.Dict, error) {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	var (
		keyType   *language.Type
		valueType *language.Type

		keys   []language.Object = make([]language.Object, 0, len(dict))
		values []language.Object = make([]language.Object, 0, len(dict))
	)

	for key, value := range dict {
		keyObj, err := language.FromValue(key, d)
		if err != nil {
			return nil, err
		}

		if keyType == nil {
			keyType = keyObj.Type()
		} else if !keyType.Compare(keyObj.Type()) {
			keyType = language.TypeAny
		}

		keys = append(keys, keyObj)

		valueObj, err := language.FromValue(value, d)
		if err != nil {
			return nil, err
		}

		if valueType == nil {
			valueType = keyObj.Type()
		} else if !valueType.Compare(valueObj.Type()) {
			valueType = language.TypeAny
		}

		values = append(values, valueObj)
	}

	return language.NewDict(keys, values, keyType, valueType, d)
}

// Float creates a language.Float poiter
func Float(f float64, dg ...*debug.Debug) *language.Float {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewFloat(f, d)
}

// Int creates a language.Int poiter
func Int(i int, dg ...*debug.Debug) *language.Int {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewInt(int64(i), d)
}

// List creates a language.List poiter
func List(l []any, dg ...*debug.Debug) (*language.List, error) {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	var (
		itemType *language.Type
		items    = make([]language.Object, len(l))
	)

	for i, item := range l {
		itemObj, err := language.FromValue(item, d)
		if err != nil {
			return nil, err
		}

		if itemType == nil {
			itemType = itemObj.Type()
		} else if !itemType.Compare(itemObj.Type()) {
			itemType = language.TypeAny
		}

		items[i] = itemObj
	}

	return language.NewList(items, itemType, d), nil
}

// Ref creates a language.Ref poiter
func Ref(o language.Object) *language.Ref {
	return language.NewRef(o)
}

// String creates a language.String poiter
func String(s string, dg ...*debug.Debug) *language.String {
	var d *debug.Debug
	if len(dg) > 0 {
		d = dg[0]
	}

	return language.NewString(s, d)
}
