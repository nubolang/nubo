package ast

import (
	"context"
	"time"

	"github.com/bndrmrtn/tea/internal/ast/astnode"
	"github.com/bndrmrtn/tea/internal/ast/parsers"
	"github.com/bndrmrtn/tea/internal/lexer"
)

type Ast struct {
	ctx         context.Context
	nodeTimeout time.Duration
}

func New(ctx context.Context, nodeTimeout ...time.Duration) *Ast {
	nt := time.Second
	if len(nodeTimeout) > 0 {
		nt = nodeTimeout[0]
	}

	return &Ast{
		ctx:         ctx,
		nodeTimeout: nt,
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
	}

	if token.Type == lexer.TokenWhiteSpace || token.Type == lexer.TokenNewLine {
		*inx++
	}

	return nil, nil
}
