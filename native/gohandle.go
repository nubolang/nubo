package native

import "github.com/nubolang/nubo/language"

type GoHandle = language.GoHandle

func NewGoHandle(data any) *GoHandle {
	return language.NewGoHandle(data, nil)
}
