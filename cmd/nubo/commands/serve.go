package commands

import (
	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/server"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve <directory>/<file.nubo>",
	Short: "Start an HTTP server to serve Nubo files",
	Run:   execServe,
}

func init() {
	// Add the prepare command to the root command
	serveCmd.PersistentFlags().String("addr", "@default", "Address to listen on")
	rootCmd.AddCommand(serveCmd)
}

func execServe(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	if len(args) > 1 {
		cmd.PrintErrln("Only one folder or file can be served")
		return
	}

	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if addr == "@default" {
		addr = config.Current.Runtime.Server.Address
	}

	folderPath := args[0]

	server, err := server.New(folderPath)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}

	if err := server.Serve(addr); err != nil {
		cmd.PrintErrln(err)
	}
}
