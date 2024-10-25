package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

const (
	longAppDesc = "Discache is a distributed cacher server."
)

var (
	version = "v0.0.1-dev"

	rootCmd = &cobra.Command{
		Long: longAppDesc,
	}
)

type flagError struct{ err error }

func (e flagError) Error() string { return e.err.Error() }

// Execute root command.
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
