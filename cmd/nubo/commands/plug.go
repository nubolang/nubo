package commands

import (
	"github.com/nubolang/nubo/plug"
	"github.com/spf13/cobra"
)

// intiCmd represents the init command
var plugCmd = &cobra.Command{
	Use:   "plug <folder>",
	Short: "Initialize a Nubo plugin",
	Run:   execPlug,
}

func init() {
	// Add the plug command to the root command
	rootCmd.AddCommand(plugCmd)
}

func execPlug(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.PrintErrln("plug requires a directory where it will create the backend")
		return
	}

	if err := plug.Scaffold(args[0]); err != nil {
		cmd.PrintErrln(err)
	}
}
