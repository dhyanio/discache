package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

const (
	appVersion  = "v0.0.1-dev"
	longAppDesc = "Discache is a distributed cache server"
)

var (
	rootCmd = &cobra.Command{
		Long: longAppDesc,
	}
)

// flagError is an error type for flag errors
type flagError struct{ err error }

func (e flagError) Error() string { return e.err.Error() }

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if !errors.As(err, &flagError{}) {
			panic(err)
		}
	}
}

func init() {
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(startCmd())
}
