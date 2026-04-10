package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var buildProject string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Query Google Cloud Build",
}

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List GCB builds (filter by commit, PR, or branch)",
	RunE: func(cmd *cobra.Command, args []string) error {
		commitSHA, _ := cmd.Flags().GetString("commit")
		prNumber, _ := cmd.Flags().GetString("pr")
		branch, _ := cmd.Flags().GetString("branch")
		limit, _ := cmd.Flags().GetInt32("limit")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).ListBuilds(context.Background(), &repodepotv1.ListBuildsRequest{
			Project:   buildProject,
			CommitSha: commitSHA,
			PrNumber:  prNumber,
			Branch:    branch,
			Limit:     limit,
		})
		if err != nil {
			return fmt.Errorf("list builds: %w", err)
		}

		if len(resp.GetBuilds()) == 0 {
			fmt.Println("no builds found")
			return nil
		}

		for _, b := range resp.GetBuilds() {
			meta := b.GetCommitSha()
			if b.GetPrNumber() != "" {
				meta += " PR#" + b.GetPrNumber()
			}

			if b.GetBranch() != "" {
				meta += " " + b.GetBranch()
			}

			fmt.Printf("%-44s %-20s %s\n", b.GetId(), b.GetStatus(), meta)
		}

		return nil
	},
}

var buildLogsCmd = &cobra.Command{
	Use:   "logs <build-id>",
	Short: "Fetch logs for a GCB build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GetBuildLogs(context.Background(), &repodepotv1.GetBuildLogsRequest{
			Project: buildProject,
			BuildId: args[0],
		})
		if err != nil {
			return fmt.Errorf("get build logs: %w", err)
		}

		fmt.Println(resp.GetLogs())

		return nil
	},
}

func init() {
	buildCmd.PersistentFlags().StringVar(&buildProject, "project", "", "GCP project (uses server default if omitted)")

	buildListCmd.Flags().String("commit", "", "Filter by commit SHA")
	buildListCmd.Flags().String("pr", "", "Filter by PR number")
	buildListCmd.Flags().String("branch", "", "Filter by branch name")
	buildListCmd.Flags().Int32("limit", 10, "Maximum number of results")

	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildLogsCmd)
	rootCmd.AddCommand(buildCmd)
}
