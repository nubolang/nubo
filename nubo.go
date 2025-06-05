package nubo

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/version"
)

const Version = version.Version

type Ctx struct {
	r *runtime.Runtime
}

func New() *Ctx {
	return NewWithProvider(events.NewDefaultProvider())
}

func NewWithProvider(provider events.Provider) *Ctx {
	return &Ctx{
		r: runtime.New(provider),
	}
}

func (c *Ctx) Exec(r io.Reader) (language.Object, error) {
	lx := lexer.New("nativeExecute")
	tokens, err := lx.Parse(r)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()

	parser := ast.New(ctx, time.Second*5)
	nodes, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}

	return c.r.Interpret("nativeExecute", nodes)
}

func (c *Ctx) ExecString(s string) (language.Object, error) {
	return c.Exec(strings.NewReader(s))
}
