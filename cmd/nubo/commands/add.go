package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a package to the current project from a remote host",
	Run:   execAdd,
}

func init() {
	// Add the add command to the root command
	rootCmd.AddCommand(addCmd)
}

func execAdd(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	p, err := packer.New(".")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	for _, repo := range args {
		if err := p.Add(repo); err != nil {
			cmd.PrintErrln(err)
			return
		}
	}
}
