package commands

import (
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/nubogo/nubo/cmd/nubo/logger"
	"github.com/nubogo/nubo/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// rootCmd is the root command for the CLI
var rootCmd = &cobra.Command{
	Use:     "nubo",
	Short:   "Nubo ☁️ A programming language built for real-time web development.",
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		nocolor, _ := cmd.Flags().GetBool("nocolor")
		color.NoColor = nocolor
		loglevel, _ := cmd.Flags().GetString("loglevel")
		dev, _ := cmd.Flags().GetBool("dev")

		os.Setenv("NUBO_DEV", strconv.FormatBool(dev))

		logger := logger.Create(loglevel)
		zap.ReplaceGlobals(logger)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolP("nocolor", "", false, "Disable colorized output")
	rootCmd.PersistentFlags().BoolP("dev", "d", false, "Run the program in debug mode")
	rootCmd.PersistentFlags().String("loglevel", "PROD", "Language tokenizer and interpreter log level")
}

func Execute() error {
	return rootCmd.Execute()
}
