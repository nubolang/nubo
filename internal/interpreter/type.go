package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/language"
)

func (i *Interpreter) stringToType(s string) (language.ObjectComplexType, error) {
	switch s {
	case "void":
		return language.TypeVoid, nil
	case "int":
		return language.TypeInt, nil
	case "string":
		return language.TypeString, nil
	case "bool":
		return language.TypeBool, nil
	case "float":
		return language.TypeFloat, nil
	case "byte":
		return language.TypeByte, nil
	case "char":
		return language.TypeChar, nil
	default:
		return nil, fmt.Errorf("unknown type: %s", s)
	}
}
