package cmd

import (
	"context"
	"fmt"
	"strconv"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	prRepo string
	prOrg  string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open a new pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		head, _ := cmd.Flags().GetString("head")
		base, _ := cmd.Flags().GetString("base")
		draft, _ := cmd.Flags().GetBool("draft")

		if title == "" {
			return fmt.Errorf("--title is required")
		}

		if head == "" {
			return fmt.Errorf("--head is required")
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).CreatePR(context.Background(), &repodepotv1.CreatePRRequest{
			Repo:  prRepo,
			Org:   prOrg,
			Title: title,
			Body:  body,
			Head:  head,
			Base:  base,
			Draft: draft,
		})
		if err != nil {
			return fmt.Errorf("create PR: %w", err)
		}

		fmt.Printf("#%d %s\n%s\n", resp.GetNumber(), resp.GetTitle(), resp.GetUrl())

		return nil
	},
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, _ := cmd.Flags().GetString("state")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).ListPRs(context.Background(), &repodepotv1.ListPRsRequest{
			Repo:  prRepo,
			Org:   prOrg,
			State: state,
		})
		if err != nil {
			return fmt.Errorf("list PRs: %w", err)
		}

		if len(resp.GetPullRequests()) == 0 {
			fmt.Println("no pull requests found")
			return nil
		}

		for _, pr := range resp.GetPullRequests() {
			fmt.Printf("#%-5d [%-6s] %s  (%s → %s)\n",
				pr.GetNumber(), pr.GetState(), pr.GetTitle(),
				pr.GetHeadBranch(), pr.GetBaseBranch())
		}

		return nil
	},
}

var prGetCmd = &cobra.Command{
	Use:   "get <number>",
	Short: "Get details for a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GetPR(context.Background(), &repodepotv1.GetPRRequest{
			Repo:   prRepo,
			Org:    prOrg,
			Number: number,
		})
		if err != nil {
			return fmt.Errorf("get PR: %w", err)
		}

		pr := resp.GetPullRequest()
		fmt.Printf("#%d %s [%s]\n", pr.GetNumber(), pr.GetTitle(), pr.GetState())
		fmt.Printf("  %s → %s\n", pr.GetHeadBranch(), pr.GetBaseBranch())
		fmt.Printf("  by %s on %s\n", pr.GetAuthor(), pr.GetCreatedAt())
		fmt.Printf("  %s\n", pr.GetUrl())

		if pr.GetBody() != "" {
			fmt.Printf("\n%s\n", pr.GetBody())
		}

		return nil
	},
}

var prMergeCmd = &cobra.Command{
	Use:   "merge <number>",
	Short: "Merge a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		method, _ := cmd.Flags().GetString("method")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).MergePR(context.Background(), &repodepotv1.MergePRRequest{
			Repo:   prRepo,
			Org:    prOrg,
			Number: number,
			Method: method,
		})
		if err != nil {
			return fmt.Errorf("merge PR: %w", err)
		}

		fmt.Printf("PR #%d merged\n", number)

		return nil
	},
}

var prRequestReviewCmd = &cobra.Command{
	Use:   "request-review <number> <reviewer> [reviewer...]",
	Short: "Request a review on a pull request",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).RequestReview(context.Background(), &repodepotv1.RequestReviewRequest{
			Repo:      prRepo,
			Org:       prOrg,
			Number:    number,
			Reviewers: args[1:],
		})
		if err != nil {
			return fmt.Errorf("request review: %w", err)
		}

		fmt.Printf("review requested on PR #%d from: %v\n", number, args[1:])

		return nil
	},
}

var prCommentsCmd = &cobra.Command{
	Use:   "comments <number>",
	Short: "List all comments on a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).ListPRComments(context.Background(), &repodepotv1.ListPRCommentsRequest{
			Repo:   prRepo,
			Org:    prOrg,
			Number: number,
		})
		if err != nil {
			return fmt.Errorf("list comments: %w", err)
		}

		if len(resp.GetComments()) == 0 {
			fmt.Println("no comments found")
			return nil
		}

		for _, c := range resp.GetComments() {
			header := fmt.Sprintf("[%s] %s (%s)", c.GetCommentType(), c.GetAuthor(), c.GetCreatedAt())

			if c.GetCommentType() == "review" && c.GetPath() != "" {
				header += fmt.Sprintf(" — %s:%d", c.GetPath(), c.GetLine())
			}

			if c.GetInReplyTo() != 0 {
				header += fmt.Sprintf(" (reply to #%d)", c.GetInReplyTo())
			}

			fmt.Printf("%s\n  id:%d  %s\n%s\n\n", header, c.GetId(), c.GetUrl(), c.GetBody())
		}

		return nil
	},
}

var prCommentCmd = &cobra.Command{
	Use:   "comment <number>",
	Short: "Post a comment on a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		body, _ := cmd.Flags().GetString("body")
		if body == "" {
			return fmt.Errorf("--body is required")
		}

		inReplyTo, _ := cmd.Flags().GetInt64("reply-to")

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).CommentOnPR(context.Background(), &repodepotv1.CommentOnPRRequest{
			Repo:      prRepo,
			Org:       prOrg,
			Number:    number,
			Body:      body,
			InReplyTo: inReplyTo,
		})
		if err != nil {
			return fmt.Errorf("post comment: %w", err)
		}

		fmt.Printf("comment posted (id: %d)\n%s\n", resp.GetCommentId(), resp.GetUrl())

		return nil
	},
}

var prChecksCmd = &cobra.Command{
	Use:   "checks <number>",
	Short: "List CI checks for a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).ListPRChecks(context.Background(), &repodepotv1.ListPRChecksRequest{
			Repo:   prRepo,
			Org:    prOrg,
			Number: number,
		})
		if err != nil {
			return fmt.Errorf("list checks: %w", err)
		}

		if len(resp.GetChecks()) == 0 {
			fmt.Println("no checks found")
			return nil
		}

		for _, c := range resp.GetChecks() {
			conclusion := c.GetConclusion()
			if conclusion == "" {
				conclusion = c.GetStatus()
			}

			fmt.Printf("%-40s %-12s run:%d\n", c.GetName(), conclusion, c.GetRunId())
		}

		return nil
	},
}

var prLogsCmd = &cobra.Command{
	Use:   "logs <run-id>",
	Short: "Get failed-step logs for a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid run ID: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GetCheckLogs(context.Background(), &repodepotv1.GetCheckLogsRequest{
			Repo:  prRepo,
			Org:   prOrg,
			RunId: runID,
		})
		if err != nil {
			return fmt.Errorf("get logs: %w", err)
		}

		fmt.Println(resp.GetLogs())

		return nil
	},
}

var prTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Fetch the PR description template for a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		resp, err := repodepotv1.NewRepodepotServiceClient(conn).GetPRTemplate(context.Background(), &repodepotv1.GetPRTemplateRequest{
			Repo: prRepo,
			Org:  prOrg,
		})
		if err != nil {
			return fmt.Errorf("get PR template: %w", err)
		}

		fmt.Print(resp.GetTemplate())

		return nil
	},
}

func init() {
	// Persistent flags shared by all pr subcommands.
	prCmd.PersistentFlags().StringVar(&prRepo, "repo", "", "Repository name (required)")
	prCmd.PersistentFlags().StringVar(&prOrg, "org", "", "GitHub organization (uses server default if omitted)")
	_ = prCmd.MarkPersistentFlagRequired("repo")

	// pr create flags.
	prCreateCmd.Flags().String("title", "", "PR title (required)")
	prCreateCmd.Flags().String("body", "", "PR body")
	prCreateCmd.Flags().String("head", "", "Head branch to merge from (required)")
	prCreateCmd.Flags().String("base", "main", "Base branch to merge into")
	prCreateCmd.Flags().Bool("draft", false, "Open as a draft PR")
	// pr list flags.
	prListCmd.Flags().String("state", "open", "Filter by state: open, closed, merged, all")

	// pr merge flags.
	prMergeCmd.Flags().String("method", "merge", "Merge method: merge, squash, rebase")

	// pr comment flags.
	prCommentCmd.Flags().String("body", "", "Comment body (required)")
	prCommentCmd.Flags().Int64("reply-to", 0, "Reply to a review comment thread (comment ID)")

	prCmd.AddCommand(prTemplateCmd)
	prCmd.AddCommand(prCreateCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prGetCmd)
	prCmd.AddCommand(prMergeCmd)
	prCmd.AddCommand(prRequestReviewCmd)
	prCmd.AddCommand(prCommentsCmd)
	prCmd.AddCommand(prCommentCmd)
	prCmd.AddCommand(prChecksCmd)
	prCmd.AddCommand(prLogsCmd)
	rootCmd.AddCommand(prCmd)
}
