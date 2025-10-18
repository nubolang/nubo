package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// intiCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a Nubo package",
	Run:   execInit,
}

func init() {
	// Add the init command to the root command
	rootCmd.AddCommand(initCmd)
}

func execInit(cmd *cobra.Command, args []string) {
	p, err := packer.Init(".")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if err := p.Write(); err != nil {
		cmd.PrintErrln(err)
		return
	}
}
