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

type gtNameInput struct {
	Name string `json:"name"`
}

type gtCreateInput struct {
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	Message string `json:"message"`
}

type gtSubmitInput struct {
	Name  string `json:"name"`
	Stack bool   `json:"stack"`
	Draft bool   `json:"draft"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type gtMoveInput struct {
	Name  string `json:"name"`
	Steps int32  `json:"steps"`
}

type gtOutput struct {
	Result string `json:"result"`
}

// --- registration ---

func registerGtTools(reg *serverproto.Registry, svc *service.RepodepotService) error {
	if err := serverproto.RegisterTool[*gtNameInput, *gtOutput](
		reg, "gt_sync",
		"Sync the Graphite stack with remote and restack as needed.",
		func(ctx context.Context, input *gtNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.GtSync(ctx, &repodepotv1.GtSyncRequest{Name: input.Name}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Graphite stack synced for project %q.", input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*gtCreateInput, *gtOutput](
		reg, "gt_create",
		"Create a new stacked branch in the project workspace.",
		func(ctx context.Context, input *gtCreateInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.GtCreate(ctx, &repodepotv1.GtCreateRequest{
				Name:    input.Name,
				Branch:  input.Branch,
				Message: input.Message,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Branch %q created in project %q.", input.Branch, input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*gtSubmitInput, *gtOutput](
		reg, "gt_submit",
		"Submit the Graphite stack (or current branch) to GitHub as PRs. Requires approval if configured.",
		func(ctx context.Context, input *gtSubmitInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GtSubmit(ctx, &repodepotv1.GtSubmitRequest{
				Name:  input.Name,
				Stack: input.Stack,
				Draft: input.Draft,
				Title: input.Title,
				Body:  input.Body,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("%s", resp.GetOutput()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*gtMoveInput, *gtOutput](
		reg, "gt_up",
		"Move up the Graphite stack by n branches (default 1).",
		func(ctx context.Context, input *gtMoveInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GtUp(ctx, &repodepotv1.GtUpRequest{
				Name:  input.Name,
				Steps: input.Steps,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Now on branch %q.", resp.GetBranch()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*gtMoveInput, *gtOutput](
		reg, "gt_down",
		"Move down the Graphite stack by n branches (default 1).",
		func(ctx context.Context, input *gtMoveInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GtDown(ctx, &repodepotv1.GtDownRequest{
				Name:  input.Name,
				Steps: input.Steps,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Now on branch %q.", resp.GetBranch()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*gtNameInput, *gtOutput](
		reg, "gt_log",
		"Show the current Graphite stack for a project.",
		func(ctx context.Context, input *gtNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.GtLog(ctx, &repodepotv1.GtLogRequest{Name: input.Name})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("%s", resp.GetOutput()), nil
		},
	); err != nil {
		return err
	}

	return nil
}
