package language

import (
	"encoding/hex"
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

type Byte struct {
	Data  byte
	debug *debug.Debug
}

func NewByte(value byte, debug *debug.Debug) *Byte {
	return &Byte{
		Data:  value,
		debug: debug,
	}
}

func (i *Byte) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Byte) Type() *Type {
	return TypeByte
}

func (i *Byte) Inspect() string {
	return fmt.Sprintf("(byte) %s", hex.Dump([]byte{i.Data}))
}

func (i *Byte) TypeString() string {
	return "<Object(byte)>"
}

func (i *Byte) String() string {
	return hex.EncodeToString([]byte{i.Data})
}

func (i *Byte) GetPrototype() Prototype {
	return nil
}

func (i *Byte) Value() any {
	return i.Data
}

func (i *Byte) Debug() *debug.Debug {
	return i.debug
}

func (i *Byte) Clone() Object {
	return NewByte(i.Data, i.debug)
}
