package math

import "github.com/nubolang/nubo/language"

func toFloat(obj language.Object) float64 {
	switch val := obj.(type) {
	case *language.Int:
		return float64(val.Value().(int64))
	case *language.Float:
		return val.Value().(float64)
	default:
		return 0
	}
}
