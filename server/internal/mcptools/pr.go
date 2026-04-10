package mcptools

import (
	"context"
	"fmt"
	"strings"

	"github.com/eberle1080/jsonrpc"
	"github.com/eberle1080/mcp-protocol/schema"
	serverproto "github.com/eberle1080/mcp-protocol/server"
	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/eberle1080/repo-depot/server/internal/service"
)

// --- input types ---

type prCreateInput struct {
	Repo  string `json:"repo"`
	Org   string `json:"org"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

type prListInput struct {
	Repo  string `json:"repo"`
	Org   string `json:"org"`
	State string `json:"state"`
}

type prGetInput struct {
	Repo   string `json:"repo"`
	Org    string `json:"org"`
	Number int64  `json:"number"`
}

type prMergeInput struct {
	Repo   string `json:"repo"`
	Org    string `json:"org"`
	Number int64  `json:"number"`
	Method string `json:"method"`
}

type prRequestReviewInput struct {
	Repo      string   `json:"repo"`
	Org       string   `json:"org"`
	Number    int64    `json:"number"`
	Reviewers []string `json:"reviewers"`
}

type prListCommentsInput struct {
	Repo   string `json:"repo"`
	Org    string `json:"org"`
	Number int64  `json:"number"`
}

type prCommentInput struct {
	Repo      string `json:"repo"`
	Org       string `json:"org"`
	Number    int64  `json:"number"`
	Body      string `json:"body"`
	InReplyTo int64  `json:"in_reply_to"`
}

type prListChecksInput struct {
	Repo   string `json:"repo"`
	Org    string `json:"org"`
	Number int64  `json:"number"`
}

type prCheckLogsInput struct {
	Repo  string `json:"repo"`
	Org   string `json:"org"`
	RunID int64  `json:"run_id"`
}

type prGetTemplateInput struct {
	Repo string `json:"repo"`
	Org  string `json:"org"`
}

type prOutput struct {
	Result string `json:"result"`
}

// --- registration ---

func registerPRTools(reg *serverproto.Registry, svc *service.RepodepotService) error {
	if err := serverproto.RegisterTool[*prCreateInput, *prOutput](
		reg, "pr_create",
		"Open a new pull request. Requires approval if configured. Omit org to use server default.",
		func(ctx context.Context, input *prCreateInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.CreatePR(ctx, &repodepotv1.CreatePRRequest{
				Repo:  input.Repo,
				Org:   input.Org,
				Title: input.Title,
				Body:  input.Body,
				Head:  input.Head,
				Base:  input.Base,
				Draft: input.Draft,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("PR #%d created: %s\n%s", resp.GetNumber(), resp.GetTitle(), resp.GetUrl()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prListInput, *prOutput](
		reg, "pr_list",
		"List pull requests for a repository. State can be: open, closed, merged, all. Omit org to use server default.",
		func(ctx context.Context, input *prListInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.ListPRs(ctx, &repodepotv1.ListPRsRequest{
				Repo:  input.Repo,
				Org:   input.Org,
				State: input.State,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			prs := resp.GetPullRequests()
			if len(prs) == 0 {
				return textResult("No pull requests found."), nil
			}
			var lines []string
			for _, pr := range prs {
				lines = append(lines, fmt.Sprintf("#%-5d [%-7s] %s (%s → %s)",
					pr.GetNumber(), pr.GetState(), pr.GetTitle(),
					pr.GetHeadBranch(), pr.GetBaseBranch()))
			}
			return textResult("%s", strings.Join(lines, "\n")), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prGetInput, *prOutput](
		reg, "pr_get",
		"Get full details for a single pull request. Omit org to use server default.",
		func(ctx context.Context, input *prGetInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GetPR(ctx, &repodepotv1.GetPRRequest{
				Repo:   input.Repo,
				Org:    input.Org,
				Number: input.Number,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			pr := resp.GetPullRequest()
			return textResult("#%d [%s] %s\nAuthor: %s\nBranch: %s → %s\nCreated: %s\nURL: %s\n\n%s",
				pr.GetNumber(), pr.GetState(), pr.GetTitle(),
				pr.GetAuthor(), pr.GetHeadBranch(), pr.GetBaseBranch(),
				pr.GetCreatedAt(), pr.GetUrl(), pr.GetBody()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prMergeInput, *prOutput](
		reg, "pr_merge",
		"Merge a pull request. Method can be: merge, squash, rebase. Requires approval if configured. Omit org to use server default.",
		func(ctx context.Context, input *prMergeInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.MergePR(ctx, &repodepotv1.MergePRRequest{
				Repo:   input.Repo,
				Org:    input.Org,
				Number: input.Number,
				Method: input.Method,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("PR #%d merged.", input.Number), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prRequestReviewInput, *prOutput](
		reg, "pr_request_review",
		"Request review from one or more GitHub users. Omit org to use server default.",
		func(ctx context.Context, input *prRequestReviewInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.RequestReview(ctx, &repodepotv1.RequestReviewRequest{
				Repo:      input.Repo,
				Org:       input.Org,
				Number:    input.Number,
				Reviewers: input.Reviewers,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Review requested from %s on PR #%d.",
				strings.Join(input.Reviewers, ", "), input.Number), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prListCommentsInput, *prOutput](
		reg, "pr_list_comments",
		"List all comments (issue and review) on a pull request. Omit org to use server default.",
		func(ctx context.Context, input *prListCommentsInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.ListPRComments(ctx, &repodepotv1.ListPRCommentsRequest{
				Repo:   input.Repo,
				Org:    input.Org,
				Number: input.Number,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			comments := resp.GetComments()
			if len(comments) == 0 {
				return textResult("No comments on PR #%d.", input.Number), nil
			}
			var lines []string
			for _, c := range comments {
				header := fmt.Sprintf("[%s] %s (%s)", c.GetCommentType(), c.GetAuthor(), c.GetCreatedAt())
				if c.GetPath() != "" {
					header += fmt.Sprintf(" — %s:%d", c.GetPath(), c.GetLine())
				}
				lines = append(lines, header)
				lines = append(lines, fmt.Sprintf("id:%d  %s", c.GetId(), c.GetUrl()))
				lines = append(lines, c.GetBody())
				lines = append(lines, "")
			}
			return textResult("%s", strings.Join(lines, "\n")), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prCommentInput, *prOutput](
		reg, "pr_comment",
		"Post a comment on a pull request. Set in_reply_to to reply to a specific review comment thread. Requires approval if configured. Omit org to use server default.",
		func(ctx context.Context, input *prCommentInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.CommentOnPR(ctx, &repodepotv1.CommentOnPRRequest{
				Repo:      input.Repo,
				Org:       input.Org,
				Number:    input.Number,
				Body:      input.Body,
				InReplyTo: input.InReplyTo,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Comment posted (id:%d): %s", resp.GetCommentId(), resp.GetUrl()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prListChecksInput, *prOutput](
		reg, "pr_list_checks",
		"List CI check statuses for a pull request. Omit org to use server default.",
		func(ctx context.Context, input *prListChecksInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.ListPRChecks(ctx, &repodepotv1.ListPRChecksRequest{
				Repo:   input.Repo,
				Org:    input.Org,
				Number: input.Number,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			checks := resp.GetChecks()
			if len(checks) == 0 {
				return textResult("No checks found for PR #%d.", input.Number), nil
			}
			var lines []string
			for _, c := range checks {
				lines = append(lines, fmt.Sprintf("%-40s %-12s run:%d  %s",
					c.GetName(), c.GetConclusion(), c.GetRunId(), c.GetUrl()))
			}
			return textResult("%s", strings.Join(lines, "\n")), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prCheckLogsInput, *prOutput](
		reg, "pr_check_logs",
		"Get failed-step logs for a GitHub Actions workflow run. Omit org to use server default.",
		func(ctx context.Context, input *prCheckLogsInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GetCheckLogs(ctx, &repodepotv1.GetCheckLogsRequest{
				Repo:  input.Repo,
				Org:   input.Org,
				RunId: input.RunID,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("%s", resp.GetLogs()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*prGetTemplateInput, *prOutput](
		reg, "pr_get_template",
		"Fetch the pull request description template from a GitHub repo. Omit org to use server default.",
		func(ctx context.Context, input *prGetTemplateInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GetPRTemplate(ctx, &repodepotv1.GetPRTemplateRequest{
				Repo: input.Repo,
				Org:  input.Org,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			tmpl := resp.GetTemplate()
			if tmpl == "" {
				return textResult("No PR template found."), nil
			}
			return textResult("%s", tmpl), nil
		},
	); err != nil {
		return err
	}

	return nil
}
