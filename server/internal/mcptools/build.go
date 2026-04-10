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

type buildListInput struct {
	Project   string `json:"project"`
	CommitSHA string `json:"commit_sha"`
	PRNumber  string `json:"pr_number"`
	Branch    string `json:"branch"`
	Limit     int32  `json:"limit"`
}

type buildLogsInput struct {
	Project string `json:"project"`
	BuildID string `json:"build_id"`
}

type buildOutput struct {
	Result string `json:"result"`
}

// --- registration ---

func registerBuildTools(reg *serverproto.Registry, svc *service.RepodepotService) error {
	if err := serverproto.RegisterTool[*buildListInput, *buildOutput](
		reg, "build_list",
		"List Google Cloud Build builds. Filter by commit SHA, PR number, or branch. Omit project to use server default.",
		func(ctx context.Context, input *buildListInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.ListBuilds(ctx, &repodepotv1.ListBuildsRequest{
				Project:   input.Project,
				CommitSha: input.CommitSHA,
				PrNumber:  input.PRNumber,
				Branch:    input.Branch,
				Limit:     input.Limit,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			builds := resp.GetBuilds()
			if len(builds) == 0 {
				return textResult("No builds found."), nil
			}
			var lines []string
			for _, b := range builds {
				line := fmt.Sprintf("%-12s %-10s %s", b.GetId(), b.GetStatus(), b.GetCreateTime())
				if b.GetCommitSha() != "" {
					line += fmt.Sprintf("  commit:%s", b.GetCommitSha()[:min(8, len(b.GetCommitSha()))])
				}
				if b.GetPrNumber() != "" {
					line += fmt.Sprintf("  pr:%s", b.GetPrNumber())
				}
				if b.GetBranch() != "" {
					line += fmt.Sprintf("  branch:%s", b.GetBranch())
				}
				lines = append(lines, line)
			}
			return textResult("%s", strings.Join(lines, "\n")), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*buildLogsInput, *buildOutput](
		reg, "build_logs",
		"Get log output for a specific Google Cloud Build. Omit project to use server default.",
		func(ctx context.Context, input *buildLogsInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GetBuildLogs(ctx, &repodepotv1.GetBuildLogsRequest{
				Project: input.Project,
				BuildId: input.BuildID,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("%s", resp.GetLogs()), nil
		},
	); err != nil {
		return err
	}

	return nil
}
