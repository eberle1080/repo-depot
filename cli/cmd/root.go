package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var serverAddr string

var rootCmd = &cobra.Command{
	Use:   "repo",
	Short: "repo-depot CLI",
	Long:  "CLI client for the repo-depot gRPC server.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:50051", "gRPC server address")
}
