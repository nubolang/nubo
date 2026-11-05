package nubo

import (
	"context"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
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
	config.Verify()

	return &Ctx{
		r: runtime.New(provider),
	}
}

// NewWithFS reads scripts from files. A nil filesystem uses the host filesystem.
func NewWithFS(files fs.FS) *Ctx {
	config.Verify()

	return &Ctx{
		r: runtime.NewWithFS(events.NewDefaultProvider(), files),
	}
}

func (c *Ctx) ExecFile(name string) (language.Object, error) {
	file, err := c.r.Source().Resolve(".", name)
	if err != nil {
		return nil, err
	}

	nodes, err := c.r.Source().Nodes(file)
	if err != nil {
		return nil, err
	}

	return c.r.Interpret(file, nodes)
}

func (c *Ctx) Tokenize(r io.Reader) ([]*lexer.Token, error) {
	lx, err := lexer.New(r, "<nativeExecute>")
	if err != nil {
		return nil, err
	}

	return lx.Parse()
}

func (c *Ctx) Parse(tokens []*lexer.Token) ([]*astnode.Node, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	defer cancel()

	parser := ast.New(ctx, time.Second*5)
	return parser.Parse(tokens)
}

func (c *Ctx) Exec(r io.Reader) (language.Object, error) {
	tokens, err := c.Tokenize(r)
	if err != nil {
		return nil, err
	}

	nodes, err := c.Parse(tokens)
	if err != nil {
		return nil, err
	}

	return c.r.Interpret("<nativeExecute>", nodes)
}

func (c *Ctx) ExecString(s string) (language.Object, error) {
	return c.Exec(strings.NewReader(s))
}
