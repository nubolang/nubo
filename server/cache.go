package server

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
	"go.uber.org/zap"
)

// CacheDuration is the duration of the cache
const CacheDuration = time.Minute * 5

// NodeCache is the cache of the nodes
type NodeCache struct {
	Expiration time.Time
	Hash       uint64
	Nodes      []*astnode.Node
}

// getCache gets the cache
func (s *Server) getCache(path string) ([]*astnode.Node, bool) {
	s.mu.RLock()
	cache, ok := s.cache[path]
	s.mu.RUnlock()
	if !ok {
		zap.L().Debug("server.cache.miss", zap.String("path", path))
		return nil, false
	}

	currentHash, err := s.hashFile(path)
	if err != nil {
		zap.L().Debug("server.cache.hashError", zap.String("path", path), zap.Error(err))
		return nil, false
	}

	if currentHash != cache.Hash {
		zap.L().Debug("server.cache.hashMismatch", zap.String("path", path))
		return nil, false
	}

	if time.Now().After(cache.Expiration) {
		zap.L().Debug("server.cache.expired", zap.String("path", path))
		return nil, false
	}

	zap.L().Debug("server.cache.hit", zap.String("path", path))
	return cache.Nodes, true
}

// setCache sets the cache
func (s *Server) setCache(path string, nodes []*astnode.Node) {
	hash, err := s.hashFile(path)
	if err != nil {
		zap.L().Warn("server.cache.set.error", zap.String("path", path), zap.Error(err))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[path] = &NodeCache{
		Expiration: time.Now().Add(CacheDuration),
		Hash:       hash,
		Nodes:      nodes,
	}
	zap.L().Debug("server.cache.set", zap.String("path", path), zap.Int("nodeCount", len(nodes)))
}

func (s *Server) hashFile(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

func (s *Server) getFile(path string) ([]*astnode.Node, bool, error) {
	if nodes, ok := s.getCache(path); ok {
		zap.L().Debug("server.cache.serve", zap.String("path", path))
		return nodes, true, nil
	}

	file, err := os.Open(path)
	if err != nil {
		zap.L().Error("server.file.open", zap.String("path", path), zap.Error(err))
		return nil, false, err
	}
	defer file.Close()

	lx, err := lexer.New(file, path)
	if err != nil {
		zap.L().Error("server.file.lexer", zap.String("path", path), zap.Error(err))
		return nil, false, err
	}
	tokens, err := lx.Parse()
	if err != nil {
		zap.L().Error("server.file.tokens", zap.String("path", path), zap.Error(err))
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	parser := ast.New(ctx, time.Second*5)
	nodes, err := parser.Parse(tokens)
	if err != nil {
		zap.L().Error("server.file.parse", zap.String("path", path), zap.Error(err))
		return nil, false, err
	}
	zap.L().Debug("server.file.ready", zap.String("path", path), zap.Int("nodes", len(nodes)), zap.Int("tokens", len(tokens)))

	s.setCache(path, nodes)
	return nodes, false, nil
}
