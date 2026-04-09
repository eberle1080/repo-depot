package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories with upstream",
}

var syncFetchCmd = &cobra.Command{
	Use:   "fetch <project> <remote-url>",
	Short: "Refresh the bare cache for a remote URL (no workspace changes)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).FetchRepo(context.Background(), &repodepotv1.FetchRepoRequest{
			Name:      args[0],
			RemoteUrl: args[1],
		})
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		fmt.Println("fetched")

		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull <project> <remote-url>",
	Short: "Pull upstream changes for a subtree into the project workspace",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		useGraphite, _ := cmd.Flags().GetBool("graphite")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).PullRepo(context.Background(), &repodepotv1.PullRepoRequest{
			Name:        args[0],
			RemoteUrl:   args[1],
			UseGraphite: useGraphite,
		})
		if err != nil {
			return fmt.Errorf("pull: %w", err)
		}

		fmt.Println("pulled")

		return nil
	},
}

var syncProjectCmd = &cobra.Command{
	Use:   "project <project>",
	Short: "Sync all subtrees in a project (fetch or pull)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fetchOnly, _ := cmd.Flags().GetBool("fetch-only")
		useGraphite, _ := cmd.Flags().GetBool("graphite")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).SyncProject(context.Background(), &repodepotv1.SyncProjectRequest{
			Name:        args[0],
			FetchOnly:   fetchOnly,
			UseGraphite: useGraphite,
		})
		if err != nil {
			return fmt.Errorf("sync project: %w", err)
		}

		fmt.Println("synced")

		return nil
	},
}

func init() {
	syncPullCmd.Flags().Bool("graphite", false, "Use gt sync instead of git rebase when on a feature branch")

	syncProjectCmd.Flags().Bool("fetch-only", false, "Only refresh bare caches; do not update workspaces")
	syncProjectCmd.Flags().Bool("graphite", false, "Use gt sync instead of git rebase when on a feature branch")

	syncCmd.AddCommand(syncFetchCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncProjectCmd)
	rootCmd.AddCommand(syncCmd)
}
