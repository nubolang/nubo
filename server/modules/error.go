package modules

import (
	"context"

	"github.com/nubolang/nubo/language"
)

func NewError(code int, message string) language.Object {
	inst := language.NewStruct("@server/error", nil, nil)
	proto := inst.GetPrototype()

	proto.SetObject(context.Background(), "status", language.NewInt(int64(code), nil))
	proto.SetObject(context.Background(), "message", language.NewString(message, nil))

	return inst
}
