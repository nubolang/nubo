package ast

import (
	"context"
	"time"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/ast/parsers"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/internal/lexer"
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
	case lexer.TokenInclude:
		node, err := parsers.IncludeParser(a.ctx, a, tokens, inx)
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
		node, err := parsers.FnParser(a.ctx, a, tokens, inx, New(a.ctx, a.nodeTimeout), false)
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
	case lexer.TokenWhile:
		node, err := parsers.WhileParser(a.ctx, a, tokens, inx)
		if err != nil {
			return nil, err
		}
		return node, nil
	case lexer.TokenIf:
		return parsers.IfParser(a.ctx, a, tokens, inx)
	case lexer.TokenFor:
		return parsers.ForParser(a.ctx, a, tokens, inx)
	case lexer.TokenImpl:
		return parsers.ImplParser(a.ctx, a, tokens, inx)
	case lexer.TokenBreak, lexer.TokenContinue:
		*inx++
		return &astnode.Node{
			Type:    astnode.NodeTypeSignal,
			Content: token.Value,
			Debug:   token.Debug,
		}, nil
	case lexer.TokenTry:
		return parsers.TryParser(a.ctx, a, tokens, inx)
	case lexer.TokenDefer:
		return parsers.DeferParser(a.ctx, a, tokens, inx)
	}

	if token.Type == lexer.TokenWhiteSpace || token.Type == lexer.TokenNewLine || token.Type == lexer.TokenSingleLineComment || token.Type == lexer.TokenMultiLineComment || token.Type == lexer.TokenSemicolon {
		*inx++
		return nil, nil
	}

	return nil, exception.Create("unhandled node: %s: '%s'", token.Type, token.Value).WithDebug(token.Debug).WithLevel(exception.LevelSemantic)
}
