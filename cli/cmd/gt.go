package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var gtCmd = &cobra.Command{
	Use:   "gt",
	Short: "Graphite stack operations (opt-in, for stacked PRs)",
}

var gtSyncCmd = &cobra.Command{
	Use:   "sync <project>",
	Short: "Sync the stack with the remote and restack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).GtSync(context.Background(), &repodepotv1.GtSyncRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("gt sync: %w", err)
		}

		fmt.Println("synced")

		return nil
	},
}

var gtCreateCmd = &cobra.Command{
	Use:   "create <project> <branch>",
	Short: "Create a new branch stacked on the current branch",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			return fmt.Errorf("--message is required")
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).GtCreate(context.Background(), &repodepotv1.GtCreateRequest{
			Name:    args[0],
			Branch:  args[1],
			Message: message,
		})
		if err != nil {
			return fmt.Errorf("gt create: %w", err)
		}

		fmt.Printf("created branch %q\n", args[1])

		return nil
	},
}

var gtSubmitCmd = &cobra.Command{
	Use:   "submit <project>",
	Short: "Submit the stack (or current branch) to GitHub as PRs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stack, _ := cmd.Flags().GetBool("stack")
		draft, _ := cmd.Flags().GetBool("draft")
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GtSubmit(context.Background(), &repodepotv1.GtSubmitRequest{
			Name:  args[0],
			Stack: stack,
			Draft: draft,
			Title: title,
			Body:  body,
		})
		if err != nil {
			return fmt.Errorf("gt submit: %w", err)
		}

		if resp.GetOutput() != "" {
			fmt.Println(resp.GetOutput())
		}

		return nil
	},
}

var gtUpCmd = &cobra.Command{
	Use:   "up <project>",
	Short: "Move up the stack by n branches (default 1)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		steps, _ := cmd.Flags().GetInt32("steps")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GtUp(context.Background(), &repodepotv1.GtUpRequest{
			Name:  args[0],
			Steps: steps,
		})
		if err != nil {
			return fmt.Errorf("gt up: %w", err)
		}

		fmt.Printf("now on %s\n", resp.GetBranch())

		return nil
	},
}

var gtDownCmd = &cobra.Command{
	Use:   "down <project>",
	Short: "Move down the stack by n branches (default 1)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		steps, _ := cmd.Flags().GetInt32("steps")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GtDown(context.Background(), &repodepotv1.GtDownRequest{
			Name:  args[0],
			Steps: steps,
		})
		if err != nil {
			return fmt.Errorf("gt down: %w", err)
		}

		fmt.Printf("now on %s\n", resp.GetBranch())

		return nil
	},
}

var gtLogCmd = &cobra.Command{
	Use:   "log <project>",
	Short: "Show the current Graphite stack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GtLog(context.Background(), &repodepotv1.GtLogRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("gt log: %w", err)
		}

		fmt.Println(resp.GetOutput())

		return nil
	},
}

func init() {
	gtCreateCmd.Flags().StringP("message", "m", "", "Commit message for the new branch (required)")

	gtSubmitCmd.Flags().Bool("stack", false, "Submit the entire stack (not just current branch)")
	gtSubmitCmd.Flags().Bool("draft", false, "Submit as draft PRs")
	gtSubmitCmd.Flags().String("title", "", "PR title for new PRs (defaults to commit message)")
	gtSubmitCmd.Flags().String("body", "", "PR description for new PRs (use 'pr template' to get the template first)")


	gtUpCmd.Flags().Int32("steps", 1, "Number of branches to move up")
	gtDownCmd.Flags().Int32("steps", 1, "Number of branches to move down")

	gtCmd.AddCommand(gtSyncCmd)
	gtCmd.AddCommand(gtCreateCmd)
	gtCmd.AddCommand(gtSubmitCmd)
	gtCmd.AddCommand(gtUpCmd)
	gtCmd.AddCommand(gtDownCmd)
	gtCmd.AddCommand(gtLogCmd)
	rootCmd.AddCommand(gtCmd)
}
