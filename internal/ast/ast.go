package ast

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/ast/parsers"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/lexer"
)

type Ast struct {
	ctx         context.Context
	nodeTimeout time.Duration
	lx          *lexer.Lexer
}

func New(ctx context.Context, nodeTimeout ...time.Duration) *Ast {
	nt := time.Second
	if len(nodeTimeout) > 0 {
		nt = nodeTimeout[0]
	}

	return &Ast{
		ctx:         ctx,
		nodeTimeout: nt,
		lx:          lexer.New("<htmlattr-parser>"),
	}
}

func (a *Ast) Parse(tokens []*lexer.Token) ([]*astnode.Node, error) {
	var (
		inx   int
		nodes []*astnode.Node
	)

	for inx < len(tokens) {
		select {
		case <-a.ctx.Done():
			return nil, nil
		default:
			node, err := a.handleToken(tokens, &inx)
			if err != nil {
				return nil, err
			}

			if node != nil {
				nodes = append(nodes, node)
			}
		}
	}

	return nodes, nil
}

func (a *Ast) handleToken(tokens []*lexer.Token, inx *int) (*astnode.Node, error) {
	token := tokens[*inx]

	switch token.Type {
	case lexer.TokenImport:
		node, err := parsers.ImportParser(a.ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenEvent:
		node, err := parsers.EventParser(a.ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenStruct:
		node, err := parsers.StructParser(a.ctx, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenFn:
		node, err := parsers.FnParser(a.ctx, tokens, inx, New(a.ctx, a.nodeTimeout))
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenIdentifier:
		node, err := parsers.IdentifierParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenConst, lexer.TokenLet:
		node, err := parsers.VariableParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenReturn:
		node, err := parsers.ReturnParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenSub:
		node, err := parsers.SubParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenPub:
		node, err := parsers.PubParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	}

	if token.Type == lexer.TokenWhiteSpace || token.Type == lexer.TokenNewLine || token.Type == lexer.TokenSingleLineComment || token.Type == lexer.TokenMultiLineComment {
		*inx++
		return nil, nil
	}

	return nil, debug.NewError(errors.New("Ast error"), fmt.Sprintf("Unhandled node: %s", token.Type), token.Debug)
}
