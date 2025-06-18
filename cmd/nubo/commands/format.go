package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/formatter"
	"github.com/nubolang/nubo/internal/lexer"
	"github.com/spf13/cobra"
)

// formatCmd represents the format command
var formatCmd = &cobra.Command{
	Use:   "format <file.nubo>|<directory>",
	Short: "Format Nubo files for better readability",
	Run:   execFormat,
}

func init() {
	// Add the format command to the root command
	rootCmd.AddCommand(formatCmd)
}

func execFormat(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	files, err := getFilesFromArgs(args)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		lx, err := lexer.New(file, path)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		tokens, err := lx.Parse()
		file.Close()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		parser := ast.New(ctx, time.Second*5)
		nodes, err := parser.Parse(tokens)

		cancel()
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		nFmt := formatter.New(nodes)
		content := nFmt.Format()

		os.WriteFile(path, []byte(content), os.ModePerm)
		fmt.Println("Formatted", path)
	}
}

func getFilesFromArgs(args []string) ([]string, error) {
	var files []string

	for _, p := range args {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			files = append(files, p)
			continue
		}

		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
