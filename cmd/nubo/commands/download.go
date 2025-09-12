package commands

import (
	"github.com/nubolang/nubo/packer"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"d"},
	Short:   "Downloads all dependencies",
	Run:     execDownload,
}

func init() {
	// Add the init command to the root command
	rootCmd.AddCommand(downloadCmd)
}

func execDownload(cmd *cobra.Command, args []string) {
	p, err := packer.New(".")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if err := p.Download(); err != nil {
		cmd.PrintErrln(err)
		return
	}
}
