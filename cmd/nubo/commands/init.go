package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// intiCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <author>",
	Short: "Initialize a Nubo package",
	Run:   execInit,
}

func init() {
	// Add the init command to the root command
	rootCmd.AddCommand(initCmd)
}

func execInit(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Help()
		return
	}

	p, err := packer.New(".")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if err := p.Init(args[0]); err != nil {
		cmd.PrintErrln(err)
		return
	}
}
