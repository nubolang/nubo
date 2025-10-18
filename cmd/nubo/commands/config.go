package commands

import (
	"fmt"

	"github.com/nubolang/nubo/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c"},
	Short:   "Returns the current configuration file's path",
	Run:     execConfig,
}

func init() {
	// Add the init command to the root command
	rootCmd.AddCommand(configCmd)
}

func execConfig(cmd *cobra.Command, args []string) {
	file, err := config.GetFile()
	if err != nil {
		cmd.PrintErrln(err)
	}
	fmt.Printf("Config file: %s\n", file)
	fmt.Println(config.Current.String())
}
