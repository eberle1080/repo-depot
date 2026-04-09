package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cloneReadOnly bool
	clonePrefix   string
)

var cloneCmd = &cobra.Command{
	Use:   "clone <project-name> <remote-url>",
	Short: "Clone a remote repo into a project as a subtree",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.CloneRepo(context.Background(), &repodepotv1.CloneRepoRequest{
			Name:      args[0],
			RemoteUrl: args[1],
			ReadOnly:  cloneReadOnly,
			Prefix:    clonePrefix,
		})
		if err != nil {
			return fmt.Errorf("clone repo: %w", err)
		}

		fmt.Printf("cloned into %s\n", resp.GetPrefix())

		return nil
	},
}

func init() {
	cloneCmd.Flags().BoolVar(&cloneReadOnly, "read-only", false, "Check out on main instead of the project branch")
	cloneCmd.Flags().StringVar(&clonePrefix, "prefix", "", "Override subtree directory name (default: repo name)")
	rootCmd.AddCommand(cloneCmd)
}
