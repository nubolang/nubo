package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bndrmrtn/tea/internal/ast"
	"github.com/bndrmrtn/tea/internal/lexer"
	"gopkg.in/yaml.v3"
)

func main() {
	fileName := "./example/import.nb"

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	lx := lexer.New(filepath.Base(fileName))
	tokens, _ := lx.Parse(file)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create("gen/lexer.yaml")
	defer out.Close()
	yaml.NewEncoder(out).Encode(tokens)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	a := ast.New(ctx)
	nodes, err := a.Parse(tokens)
	if err != nil {
		log.Fatal(err)
	}

	aout, _ := os.Create("gen/ast.yaml")
	defer aout.Close()
	yaml.NewEncoder(aout).Encode(nodes)
}
