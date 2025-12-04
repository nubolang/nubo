package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <url>",
	Short: "Add a package to the current project from a remote host",
	Run:   execGet,
}

func init() {
	getCmd.Flags().BoolP("force", "f", false, "Keep going even if a package cannot be downloaded")
	getCmd.Flags().BoolP("skip-init", "s", false, "Skip initializing the package information")
	// Add the get command to the root command
	rootCmd.AddCommand(getCmd)
}

func execGet(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	force, _ := cmd.Flags().GetBool("force")
	skipInit, _ := cmd.Flags().GetBool("skip-init")

	p, err := packer.New(".", !skipInit)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	for _, repo := range args {
		if err := p.Add(repo); err != nil {
			cmd.PrintErrln(err)
			if !force {
				return
			}
		}
	}
}
