package mcptools

import (
	"context"

	"github.com/eberle1080/jsonrpc"
	"github.com/eberle1080/mcp-protocol/schema"
	serverproto "github.com/eberle1080/mcp-protocol/server"
	"github.com/eberle1080/repo-depot/server/internal/service"
	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
)

// --- input types ---

type syncRepoInput struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
}

type syncPullInput struct {
	Name        string `json:"name"`
	RemoteURL   string `json:"remote_url"`
	UseGraphite bool   `json:"use_graphite"`
}

type syncProjectInput struct {
	Name        string `json:"name"`
	FetchOnly   bool   `json:"fetch_only"`
	UseGraphite bool   `json:"use_graphite"`
}

type syncOutput struct {
	Result string `json:"result"`
}

// --- registration ---

func registerSyncTools(reg *serverproto.Registry, svc *service.RepodepotService) error {
	if err := serverproto.RegisterTool[*syncRepoInput, *syncOutput](
		reg, "sync_fetch",
		"Refresh the bare cache for a remote URL without touching any workspace.",
		func(ctx context.Context, input *syncRepoInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.FetchRepo(ctx, &repodepotv1.FetchRepoRequest{
				Name:      input.Name,
				RemoteUrl: input.RemoteURL,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Fetched %q for project %q.", input.RemoteURL, input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*syncPullInput, *syncOutput](
		reg, "sync_pull",
		"Pull upstream changes for a subtree into the project workspace. Set use_graphite=true to use gt sync instead of git rebase.",
		func(ctx context.Context, input *syncPullInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.PullRepo(ctx, &repodepotv1.PullRepoRequest{
				Name:        input.Name,
				RemoteUrl:   input.RemoteURL,
				UseGraphite: input.UseGraphite,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Pulled %q into project %q.", input.RemoteURL, input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*syncProjectInput, *syncOutput](
		reg, "sync_project",
		"Sync all subtrees in a project. Set fetch_only=true to only refresh caches without updating workspaces.",
		func(ctx context.Context, input *syncProjectInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.SyncProject(ctx, &repodepotv1.SyncProjectRequest{
				Name:        input.Name,
				FetchOnly:   input.FetchOnly,
				UseGraphite: input.UseGraphite,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			if input.FetchOnly {
				return textResult("All caches fetched for project %q.", input.Name), nil
			}
			return textResult("Project %q synced.", input.Name), nil
		},
	); err != nil {
		return err
	}

	return nil
}
