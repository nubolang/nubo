package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/config"
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
			if err := writeDebug(filePath, tokens, true); err != nil {
				cmd.PrintErrln(err)
				return
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(config.Current.Syntax.Tokenizer.Context.Deadline))
		defer cancel()

		builder := ast.New(ctx, time.Millisecond*time.Duration(config.Current.Syntax.Tokenizer.Context.Deadline))
		syntaxTree, err = builder.Parse(tokens)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		if dev {
			if err := writeDebug(filePath, syntaxTree, false); err != nil {
				cmd.PrintErrln(err)
				return
			}
		}
	}

	var eventProvider events.Provider
	if config.Current.Runtime.Events.Enabled {
		eventProvider = events.NewDefaultProvider()
	}

	ex := runtime.New(eventProvider)
	if _, err := ex.Interpret(filePath, syntaxTree); err != nil {
		cmd.PrintErrln(err)
		return
	}
}

func writeDebug(filename string, data any, lx bool) error {
	var path string
	if lx {
		path = config.Current.Syntax.Lexer.Debug.File
	} else {
		path = config.Current.Syntax.Tokenizer.Debug.File
	}

	if path == "{nubo_dir_full_file}" {
		path = filepath.Join(config.Base, "debug", filename)
	}

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(data)
}
