package language

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type GoHandle struct {
	RealData any
	debug    *debug.Debug
}

func NewGoHandle(data any, debug *debug.Debug) *GoHandle {
	return &GoHandle{RealData: data, debug: debug}
}

func (h *GoHandle) ID() string {
	return fmt.Sprintf("%p", h)
}

func (h *GoHandle) Type() *Type {
	return withObject(h, TypeGoHandle)
}

func (h *GoHandle) Inspect() string {
	return h.String()
}

func (h *GoHandle) TypeString() string {
	return "<Object(native.GoHandle)>"
}

func (h *GoHandle) String() string {
	return fmt.Sprintf("<GoHandle %p>", h)
}

func (h *GoHandle) GetPrototype() Prototype {
	return nil
}

func (h *GoHandle) Value() any {
	return h
}

func (h *GoHandle) Debug() *debug.Debug {
	return h.debug
}

func (h *GoHandle) Clone() Object {
	return h
}
