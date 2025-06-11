package sql

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

type SQLConn struct {
	Driver string
	DB     *sql.DB
	debug  *debug.Debug
}

func NewSQLConn(driver string, db *sql.DB, debug *debug.Debug) *SQLConn {
	return &SQLConn{
		Driver: driver,
		DB:     db,
		debug:  debug,
	}
}

func (i *SQLConn) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *SQLConn) Type() *language.Type {
	return n.TStruct
}

func (i *SQLConn) Inspect() string {
	return fmt.Sprintf("<Object(*SQLConn @ %s)>", strconv.Quote(i.String()))
}

func (i *SQLConn) TypeString() string {
	return "<Object(*SQLConn)>"
}

func (i *SQLConn) String() string {
	return i.Driver
}

func (i *SQLConn) GetPrototype() language.Prototype {
	return nil
}

func (i *SQLConn) Value() any {
	return i.DB
}

func (i *SQLConn) Debug() *debug.Debug {
	return i.debug
}

func (i *SQLConn) Clone() language.Object {
	return i
}
