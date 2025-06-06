package builtin

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
)

func GetBuiltins() map[string]language.Object {
	return map[string]language.Object{
		"_id":     native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, idFn),
		"println": native.NewFunction(printlnFn),
		"print":   native.NewFunction(printFn),
		"type":    native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, typeFn),
		"inspect": native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, inspectFn),
		"sleep":   native.NewTypedFunction(native.OneArg("ms", language.TypeInt, language.NewInt(0, nil)), language.TypeVoid, sleepFn),
		"ref":     native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeAny, refFn),
		"unwrap":  native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeAny, unwrapFn),
		"clone":   native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeAny, cloneFn),
		"exit":    native.NewTypedFunction(native.OneArg("code", language.TypeInt, language.NewInt(0, nil)), language.TypeVoid, exitFn),

		// Types
		"string": native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeString, stringFn),
		"int":    native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeInt, intFn),
		"float":  native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeFloat, floatFn),
		"bool":   native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeBool, boolFn),
		"byte":   native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeByte, byteFn),
		"char":   native.NewTypedFunction(native.OneArg("obj", language.TypeAny), language.TypeChar, charFn),
	}
}

func idFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.ID(), nil), nil
}

func printlnFn(args []language.Object) (language.Object, error) {
	var out []string
	for _, arg := range args {
		out = append(out, arg.String())
	}
	fmt.Println(strings.Join(out, " "))
	return nil, nil
}

func printFn(args []language.Object) (language.Object, error) {
	var out []string
	for _, arg := range args {
		out = append(out, arg.String())
	}
	fmt.Print(strings.Join(out, " "))
	return nil, nil
}

func typeFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.TypeString(), nil), nil
}

func inspectFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.Inspect(), nil), nil
}

func sleepFn(ctx native.FnCtx) (language.Object, error) {
	ms, err := ctx.Get("ms")
	if err != nil {
		return nil, err
	}

	value := ms.Value().(int64)
	if value < 0 {
		return nil, fmt.Errorf("duration must be non-negative")
	}

	time.Sleep(time.Duration(value) * time.Millisecond)
	return nil, nil
}

func stringFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}
	return language.NewString(obj.String(), nil), nil
}

func intFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	var (
		value      int64
		convertErr = debug.NewError(fmt.Errorf("Type error"), fmt.Sprintf("Cannot convert %s to int", obj.Type()), obj.Debug())
	)

	switch obj.Type() {
	case language.TypeInt:
		value = obj.Value().(int64)
	case language.TypeFloat:
		value = int64(obj.Value().(float64))
	case language.TypeBool:
		if obj.Value().(bool) {
			value = 1
		}
		break
	case language.TypeChar:
		value = int64(obj.Value().(rune))
	case language.TypeByte:
		value = int64(obj.Value().(byte))
	case language.TypeString:
		strVal := obj.Value().(string)
		val, err := strconv.Atoi(strVal)
		if err != nil {
			fval, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				return nil, convertErr
			}
			value = int64(fval)
		} else {
			value = int64(val)
		}
		break
	default:
		return nil, convertErr
	}

	return language.NewInt(value, obj.Debug()), nil
}

func floatFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	var (
		value      float64
		convertErr = debug.NewError(fmt.Errorf("Type error"), fmt.Sprintf("Cannot convert %s to float", obj.Type()), obj.Debug())
	)

	switch obj.Type() {
	case language.TypeFloat:
		value = obj.Value().(float64)
	case language.TypeInt:
		value = float64(obj.Value().(int64))
	case language.TypeBool:
		if obj.Value().(bool) {
			value = 1.0
		} else {
			value = 0.0
		}
	case language.TypeChar:
		value = float64(obj.Value().(rune))
	case language.TypeByte:
		value = float64(obj.Value().(byte))
	case language.TypeString:
		strVal := obj.Value().(string)
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return nil, convertErr
		}
		value = val
	default:
		return nil, convertErr
	}

	return language.NewFloat(value, obj.Debug()), nil
}

func boolFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	var value bool
	convertErr := debug.NewError(fmt.Errorf("Type error"), fmt.Sprintf("Cannot convert %s to bool", obj.Type()), obj.Debug())

	switch obj.Type() {
	case language.TypeBool:
		value = obj.Value().(bool)
	case language.TypeInt:
		value = obj.Value().(int64) != 0
	case language.TypeFloat:
		value = obj.Value().(float64) != 0
	case language.TypeChar:
		value = obj.Value().(rune) != 0
	case language.TypeByte:
		value = obj.Value().(byte) != 0
	case language.TypeString:
		strVal := strings.ToLower(strings.TrimSpace(obj.Value().(string)))
		if strVal == "true" || strVal == "1" {
			value = true
		} else if strVal == "false" || strVal == "0" || strVal == "" {
			value = false
		} else {
			return nil, convertErr
		}
	default:
		return nil, convertErr
	}

	return language.NewBool(value, obj.Debug()), nil
}

func byteFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	var value byte
	convertErr := debug.NewError(fmt.Errorf("Type error"), fmt.Sprintf("Cannot convert %s to byte", obj.Type()), obj.Debug())

	switch obj.Type() {
	case language.TypeByte:
		value = obj.Value().(byte)
	case language.TypeChar:
		r := obj.Value().(rune)
		if r > 255 {
			return nil, convertErr
		}
		value = byte(r)
	case language.TypeInt:
		i := obj.Value().(int64)
		if i < 0 || i > 255 {
			return nil, convertErr
		}
		value = byte(i)
	case language.TypeFloat:
		f := obj.Value().(float64)
		if f < 0 || f > 255 {
			return nil, convertErr
		}
		value = byte(int64(f))
	case language.TypeBool:
		if obj.Value().(bool) {
			value = 1
		} else {
			value = 0
		}
	case language.TypeString:
		str := obj.Value().(string)
		if len(str) != 1 {
			return nil, convertErr
		}
		value = str[0]
	default:
		return nil, convertErr
	}

	return language.NewByte(value, obj.Debug()), nil
}

func charFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	var value rune
	convertErr := debug.NewError(fmt.Errorf("Type error"), fmt.Sprintf("Cannot convert %s to char", obj.Type()), obj.Debug())

	switch obj.Type() {
	case language.TypeChar:
		value = obj.Value().(rune)
	case language.TypeByte:
		value = rune(obj.Value().(byte))
	case language.TypeInt:
		i := obj.Value().(int64)
		if i < 0 || i > 0x10FFFF { // Unicode range
			return nil, convertErr
		}
		value = rune(i)
	case language.TypeFloat:
		f := obj.Value().(float64)
		if f < 0 || f > 0x10FFFF {
			return nil, convertErr
		}
		value = rune(int64(f))
	case language.TypeBool:
		if obj.Value().(bool) {
			value = '1'
		} else {
			value = '0'
		}
	case language.TypeString:
		str := obj.Value().(string)
		runes := []rune(str)
		if len(runes) != 1 {
			return nil, convertErr
		}
		value = runes[0]
	default:
		return nil, convertErr
	}

	return language.NewChar(value, obj.Debug()), nil
}

func cloneFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	return obj.Clone(), nil
}

func exitFn(ctx native.FnCtx) (language.Object, error) {
	code, err := ctx.Get("code")
	if err != nil {
		return nil, err
	}

	os.Exit(int(code.Value().(int64)))

	return nil, nil
}

func refFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	return language.NewRef(obj), nil
}

func unwrapFn(ctx native.FnCtx) (language.Object, error) {
	obj, err := ctx.Get("obj")
	if err != nil {
		return nil, err
	}

	fmt.Printf("%T\n", obj)
	if ref, ok := obj.(*language.Ref); ok {
		return ref.Data, nil
	}

	return nil, fmt.Errorf("Cannot unwrap non-reference object")
}
