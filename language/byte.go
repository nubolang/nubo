package language

import (
	"encoding/hex"
	"fmt"

	"github.com/nubogo/nubo/internal/debug"
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

func (i *Byte) Type() ObjectComplexType {
	return TypeByte
}

func (i *Byte) Inspect() string {
	return fmt.Sprintf("<Object(byte @ %s)>", hex.Dump([]byte{i.Data}))
}

func (i *Byte) TypeString() string {
	return "<Object(byte)>"
}

func (i *Byte) String() string {
	return hex.Dump([]byte{i.Data})
}

func (i *Byte) GetPrototype() Prototype {
	return nil
}

func (i *Byte) Value() byte {
	return i.Data
}

func (i *Byte) Debug() *debug.Debug {
	return i.debug
}

func (i *Byte) Clone() *Byte {
	return NewByte(i.Data, i.debug)
}
