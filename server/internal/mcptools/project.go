package mcptools

import (
	"context"
	"strings"

	"github.com/eberle1080/jsonrpc"
	"github.com/eberle1080/mcp-protocol/schema"
	serverproto "github.com/eberle1080/mcp-protocol/server"
	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/eberle1080/repo-depot/server/internal/service"
)

// --- input/output types ---

type projectNameInput struct {
	Name string `json:"name"`
}

type projectNameOutput struct {
	Name string `json:"name"`
}

type projectRenameInput struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

type projectRenameOutput struct {
	Message string `json:"message"`
}

type repoCloneInput struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remote_url"`
	Prefix    string `json:"prefix"`
	ReadOnly  bool   `json:"read_only"`
}

type repoCloneOutput struct {
	Prefix string `json:"prefix"`
}

type emptyInput struct{}
type emptyOutput struct{}

// --- registration ---

func registerProjectTools(reg *serverproto.Registry, svc *service.RepodepotService) error {
	if err := serverproto.RegisterTool[*projectNameInput, *projectNameOutput](
		reg, "project_create",
		"Create a new project workspace with a bare repo in the depot.",
		func(ctx context.Context, input *projectNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.CreateProject(ctx, &repodepotv1.CreateProjectRequest{Name: input.Name})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Project %q created.", resp.GetName()), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*emptyInput, *emptyOutput](
		reg, "project_list",
		"List all projects in the depot.",
		func(ctx context.Context, _ *emptyInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.ListProjects(ctx, &repodepotv1.ListProjectsRequest{})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			if len(resp.GetProjects()) == 0 {
				return textResult("No projects found."), nil
			}
			var names []string
			for _, p := range resp.GetProjects() {
				names = append(names, p.GetName())
			}
			return textResult("%s", strings.Join(names, "\n")), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*projectNameInput, *emptyOutput](
		reg, "project_save",
		"Save a project: push all read-write subtrees back to bare caches and commit project state.",
		func(ctx context.Context, input *projectNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.SaveProject(ctx, &repodepotv1.SaveProjectRequest{Name: input.Name}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Project %q saved.", input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*projectNameInput, *emptyOutput](
		reg, "project_delete",
		"Archive and remove a project workspace. The workspace is archived before deletion.",
		func(ctx context.Context, input *projectNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.DeleteProject(ctx, &repodepotv1.DeleteProjectRequest{Name: input.Name}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Project %q deleted and archived.", input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*projectNameInput, *emptyOutput](
		reg, "project_checkout",
		"Recreate the workspace for an existing project from its bare repo.",
		func(ctx context.Context, input *projectNameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.CheckoutProject(ctx, &repodepotv1.CheckoutProjectRequest{Name: input.Name}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Project %q checked out.", input.Name), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*projectRenameInput, *projectRenameOutput](
		reg, "project_rename",
		"Rename a project's bare repo in the depot.",
		func(ctx context.Context, input *projectRenameInput) (*schema.CallToolResult, *jsonrpc.Error) {
			if _, err := svc.RenameProject(ctx, &repodepotv1.RenameProjectRequest{
				OldName: input.OldName,
				NewName: input.NewName,
			}); err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Project renamed from %q to %q.", input.OldName, input.NewName), nil
		},
	); err != nil {
		return err
	}

	if err := serverproto.RegisterTool[*repoCloneInput, *repoCloneOutput](
		reg, "repo_clone",
		"Clone a remote repository as a subtree into a project. Use read_only=true for dependencies you won't modify.",
		func(ctx context.Context, input *repoCloneInput) (*schema.CallToolResult, *jsonrpc.Error) {
			resp, err := svc.CloneRepo(ctx, &repodepotv1.CloneRepoRequest{
				Name:      input.Name,
				RemoteUrl: input.RemoteURL,
				Prefix:    input.Prefix,
				ReadOnly:  input.ReadOnly,
			})
			if err != nil {
				return nil, jsonrpcError(err)
			}
			return textResult("Repository cloned into project %q at prefix %q.", input.Name, resp.GetPrefix()), nil
		},
	); err != nil {
		return err
	}

	return nil
}
