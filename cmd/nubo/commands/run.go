package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/dotfolder"
	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func execRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	filePath := args[0]
	dev := os.Getenv("NUBO_DEV") == "true"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cmd.PrintErrln("File does not exist")
		return
	}

	var syntaxTree []*astnode.Node
	if prepared, ok := dotfolder.HasPrepared(filePath); ok {
		fmt.Println(color.New(color.FgBlue).Sprint("Using a prepared file 🚀"))
		syntaxTree = prepared
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		defer file.Close()

		lx, err := lexer.New(file, filePath)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		tokens, err := lx.Parse()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		if dev {
			if err := writeDebug("lexer.yaml", tokens); err != nil {
				cmd.PrintErrln(err)
				return
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		builder := ast.New(ctx, time.Second*5)
		syntaxTree, err = builder.Parse(tokens)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		if dev {
			if err := writeDebug("ast.yaml", syntaxTree); err != nil {
				cmd.PrintErrln(err)
				return
			}
		}
	}

	eventProvider := events.NewDefaultProvider()

	ex := runtime.New(eventProvider)
	if _, err := ex.Interpret(filePath, syntaxTree); err != nil {
		cmd.PrintErrln(err)
		return
	}
}

func writeDebug(filename string, data any) error {
	const dir = ".nubo/debug"
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(dir, filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(data)
}
