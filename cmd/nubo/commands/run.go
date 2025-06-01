package commands

import (
	"context"
	"os"
	"time"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/lexer"
	"github.com/nubolang/nubo/internal/pubsub"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <file.nubo>",
	Short: "Interpret and execute Nubo files",
	Run:   execRun,
}

func init() {
	// Add the run command to the root command
	rootCmd.AddCommand(runCmd)
}

func execRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	if len(args) > 1 {
		cmd.PrintErrln("Only one file can be run at a time")
		return
	}

	filePath := args[0]
	dev := os.Getenv("NUBO_DEV") == "true"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cmd.PrintErrln("File does not exist")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}
	defer file.Close()

	lx := lexer.New(filePath)
	tokens, err := lx.Parse(file)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if dev {
		lxFile, err := os.Create("./bin/gen/lexer.yaml")
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		defer lxFile.Close()

		if err := yaml.NewEncoder(lxFile).Encode(tokens); err != nil {
			cmd.PrintErrln(err)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	builder := ast.New(ctx, time.Second*5)
	syntaxTree, err := builder.Parse(tokens)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if dev {
		astFile, err := os.Create("./bin/gen/ast.yaml")
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		defer astFile.Close()

		if err := yaml.NewEncoder(astFile).Encode(syntaxTree); err != nil {
			cmd.PrintErrln(err)
			return
		}
	}

	eventProvider := pubsub.NewDefaultProvider()

	ex := runtime.New(eventProvider)
	if _, err := ex.Interpret(filePath, syntaxTree); err != nil {
		cmd.PrintErrln(err)
		return
	}
}
