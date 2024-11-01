package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// version is the version of the application
func versionCmd() *cobra.Command {
	command := cobra.Command{
		Use:   "version",
		Short: "Print version/build info",
		Long:  "Print version/build information",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	}

	return &command
}

func printVersion() {
	fmt.Println("discache version", appVersion, runtime.GOOS)
}
