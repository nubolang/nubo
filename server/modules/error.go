package modules

import (
	"github.com/nubolang/nubo/language"
)

func NewError(code int, message string) language.Object {
	inst := language.NewStruct("@server/error", nil, nil)
	proto := inst.GetPrototype()

	proto.SetObject("status", language.NewInt(int64(code), nil))
	proto.SetObject("message", language.NewString(message, nil))

	return inst
}
