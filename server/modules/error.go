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

	ctx := context.Background()
	if err := proto.SetObject(ctx, "status", language.NewInt(int64(code), nil)); err != nil {
		return nil, err
	}

	if err := proto.SetObject(ctx, "message", language.NewString(message, nil)); err != nil {
		return nil, err
	}

	return inst, nil
}
