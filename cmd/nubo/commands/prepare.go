package commands

import (
	"encoding/gob"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/dotfolder"
	"github.com/spf13/cobra"
)

// prepareCmd represents the prepare command
var prepareCmd = &cobra.Command{
	Use:   "prepare <file.nubo>",
	Short: "Prepare Nubo files for faster execution",
	Run:   execPrepare,
}

func init() {
	// Add the prepare command to the root command
	rootCmd.AddCommand(prepareCmd)
	gob.Register(&astnode.Node{})
	gob.Register(&astnode.ForValue{})
}

func execPrepare(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	if len(args) > 1 {
		cmd.PrintErrln("Only one folder can be prepared at a time")
		return
	}

	folderPath := args[0]

	if err := dotfolder.PrepareFiles(folderPath, false); err != nil {
		cmd.PrintErrln(err)
	}
}
