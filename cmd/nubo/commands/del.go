package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// delCmd represents the del command
var delCmd = &cobra.Command{
	Use:   "del <url>",
	Short: "Delete a package from the current project",
	Run:   execDel,
}

func init() {
	delCmd.Flags().BoolP("force", "f", false, "Keep going even if a package cannot be deleted")
	delCmd.Flags().Bool("cleanup", false, "Clean the package from disk after deletion")
	// Add the del command to the root command
	rootCmd.AddCommand(delCmd)
}

func execDel(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	p, err := packer.New(".")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	force, _ := cmd.Flags().GetBool("force")
	cleanUp, _ := cmd.Flags().GetBool("cleanup")

	for _, repo := range args {
		if err := p.Del(repo, cleanUp); err != nil {
			cmd.PrintErrln(err)
			if !force {
				return
			}
		}
	}
}
