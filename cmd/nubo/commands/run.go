package commands

import (
	"os"

	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/runner"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/spf13/cobra"
)

func execRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cmd.PrintErrln("File does not exist")
		return
	}

	var eventProvider events.Provider
	if config.Current.Runtime.Events.Enabled {
		eventProvider = events.NewDefaultProvider()
	}

	ex := runtime.New(eventProvider)
	ret, err := runner.Execute(filePath, ex)
	if err != nil {
		cmd.PrintErrln(err)
		return
	}
	if ret != nil {
		cmd.Println(ret.String())
	}
}
