package modules

import (
	"context"
	"fmt"

	"github.com/nubolang/nubo/language"
)

func NewError(code int, message string) (language.Object, error) {
	definition := language.NewStruct("@server/error", []language.StructField{
		{Name: "status", Type: language.TypeInt},
		{Name: "message", Type: language.TypeString},
	}, nil)

	inst, err := definition.NewInstance()
	if err != nil {
		return nil, err
	}

	proto := inst.GetPrototype()
	if proto == nil {
		return nil, fmt.Errorf("@server/error has no prototype")
	}

	proto.SetObject(context.Background(), "status", language.NewInt(int64(code), nil))
	proto.SetObject(context.Background(), "message", language.NewString(message, nil))

	return inst, nil
}
