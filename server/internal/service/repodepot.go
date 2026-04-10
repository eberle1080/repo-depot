package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/amp-labs/amp-common/logger"
	"github.com/eberle1080/repo-depot/server/config"
	"github.com/eberle1080/repo-depot/server/internal/approval"
	"github.com/eberle1080/repo-depot/server/internal/depot"
	"github.com/eberle1080/repo-depot/server/internal/gcloud"
	"github.com/eberle1080/repo-depot/server/internal/gh"
	"github.com/eberle1080/repo-depot/server/internal/gt"
	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RepodepotService implements the RepodepotServiceServer interface.
type RepodepotService struct {
	repodepotv1.UnimplementedRepodepotServiceServer
	cfg       *config.Config
	approvals *approval.Manager
}

// New creates a new RepodepotService.
func New(cfg *config.Config, approvals *approval.Manager) *RepodepotService {
	return &RepodepotService{cfg: cfg, approvals: approvals}
}

// approve gates an operation behind a human approval if the approval manager
// is configured. If RabbitMQ is not configured, the operation proceeds without
// approval. Returns a gRPC PermissionDenied error if denied or timed out.
func (s *RepodepotService) approve(ctx context.Context, title, body string) error {
	if s.approvals == nil {
		return nil
	}

	if err := s.approvals.Request(ctx, title, body); err != nil {
		return status.Errorf(codes.PermissionDenied, "approval: %v", err)
	}

	return nil
}

func (s *RepodepotService) depot() *depot.Depot {
	return depot.New(s.cfg.Git.DepotPath, s.cfg.Git.WorkspacesPath)
}

// Ping responds to a ping request.
func (s *RepodepotService) Ping(_ context.Context, req *repodepotv1.PingRequest) (*repodepotv1.PingResponse, error) {
	msg := req.GetMessage()
	if msg == "" {
		msg = "pong"
	}

	return &repodepotv1.PingResponse{Message: msg}, nil
}

// CreateProject initializes a new project workspace.
func (s *RepodepotService) CreateProject(ctx context.Context, req *repodepotv1.CreateProjectRequest) (*repodepotv1.CreateProjectResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.depot().CreateProject(ctx, name); err != nil {
		log.Error("CreateProject failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "create project: %v", err)
	}

	log.Info("project created", "name", name)

	return &repodepotv1.CreateProjectResponse{Name: name}, nil
}

// CloneRepo pulls a remote repository into a project as a subtree.
func (s *RepodepotService) CloneRepo(ctx context.Context, req *repodepotv1.CloneRepoRequest) (*repodepotv1.CloneRepoResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	remoteURL := req.GetRemoteUrl()
	if remoteURL == "" {
		return nil, status.Error(codes.InvalidArgument, "remote_url is required")
	}

	prefix := req.GetPrefix()
	if prefix == "" {
		prefix = repoName(remoteURL)
	}

	prefix, err := s.depot().CloneRepo(ctx, name, remoteURL, prefix, req.GetReadOnly())
	if err != nil {
		log.Error("CloneRepo failed", "name", name, "remote", remoteURL, "error", err)
		return nil, status.Errorf(codes.Internal, "clone repo: %v", err)
	}

	log.Info("repo cloned into project", "name", name, "remote", remoteURL, "prefix", prefix, "read_only", req.GetReadOnly())

	return &repodepotv1.CloneRepoResponse{Prefix: prefix}, nil
}

// SaveProject commits and pushes all rw subtrees and the project repo.
func (s *RepodepotService) SaveProject(ctx context.Context, req *repodepotv1.SaveProjectRequest) (*repodepotv1.SaveProjectResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.depot().SaveProject(ctx, name); err != nil {
		log.Error("SaveProject failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "save project: %v", err)
	}

	log.Info("project saved", "name", name)

	return &repodepotv1.SaveProjectResponse{}, nil
}

// ListProjects returns all projects known to the depot.
func (s *RepodepotService) ListProjects(_ context.Context, _ *repodepotv1.ListProjectsRequest) (*repodepotv1.ListProjectsResponse, error) {
	names, err := s.depot().ListProjects()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list projects: %v", err)
	}

	projects := make([]*repodepotv1.ProjectInfo, len(names))
	for i, name := range names {
		projects[i] = &repodepotv1.ProjectInfo{Name: name}
	}

	return &repodepotv1.ListProjectsResponse{Projects: projects}, nil
}

// DeleteProject archives and removes a project workspace.
func (s *RepodepotService) DeleteProject(ctx context.Context, req *repodepotv1.DeleteProjectRequest) (*repodepotv1.DeleteProjectResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	archivePath, err := s.depot().DeleteProject(ctx, name)
	if err != nil {
		log.Error("DeleteProject failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "delete project: %v", err)
	}

	log.Info("project deleted", "name", name, "archive", archivePath)

	return &repodepotv1.DeleteProjectResponse{ArchivePath: archivePath}, nil
}

// RenameProject renames the project bare repo and returns the new bare path.
func (s *RepodepotService) RenameProject(ctx context.Context, req *repodepotv1.RenameProjectRequest) (*repodepotv1.RenameProjectResponse, error) {
	log := logger.Get(ctx)

	oldName := req.GetOldName()
	newName := req.GetNewName()

	if oldName == "" {
		return nil, status.Error(codes.InvalidArgument, "old_name is required")
	}

	if newName == "" {
		return nil, status.Error(codes.InvalidArgument, "new_name is required")
	}

	newBarePath, err := s.depot().RenameProject(ctx, oldName, newName)
	if err != nil {
		log.Error("RenameProject failed", "old", oldName, "new", newName, "error", err)
		return nil, status.Errorf(codes.Internal, "rename project: %v", err)
	}

	log.Info("project renamed", "old", oldName, "new", newName)

	return &repodepotv1.RenameProjectResponse{NewBarePath: newBarePath}, nil
}

// CheckoutProject re-creates the workspace for an existing project.
func (s *RepodepotService) CheckoutProject(ctx context.Context, req *repodepotv1.CheckoutProjectRequest) (*repodepotv1.CheckoutProjectResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.depot().CheckoutProject(ctx, name); err != nil {
		log.Error("CheckoutProject failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "checkout project: %v", err)
	}

	log.Info("project checked out", "name", name)

	return &repodepotv1.CheckoutProjectResponse{}, nil
}

// resolveOrg returns the org from the request, falling back to the configured default.
func (s *RepodepotService) resolveOrg(requestOrg string) (string, error) {
	if requestOrg != "" {
		return requestOrg, nil
	}

	if s.cfg.GitHub.DefaultOrg == "" {
		return "", status.Error(codes.InvalidArgument, "org is required (no github.default_org configured)")
	}

	return s.cfg.GitHub.DefaultOrg, nil
}

// ghClient returns a gh.Runner for the given org and repo.
func (s *RepodepotService) ghClient(org, repo string) *gh.Runner {
	return gh.New(org + "/" + repo)
}

// --- PR handlers ---

// CreatePR opens a new pull request.
func (s *RepodepotService) CreatePR(ctx context.Context, req *repodepotv1.CreatePRRequest) (*repodepotv1.CreatePRResponse, error) {
	log := logger.Get(ctx)

	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	if req.GetHead() == "" {
		return nil, status.Error(codes.InvalidArgument, "head is required")
	}

	if err := s.approve(ctx,
		"Approve: Create PR?",
		fmt.Sprintf("%q — %s/%s: %s → %s", req.GetTitle(), org, repo, req.GetHead(), req.GetBase()),
	); err != nil {
		return nil, err
	}

	number, url, err := s.ghClient(org, repo).CreatePR(ctx,
		req.GetTitle(), req.GetBody(), req.GetHead(), req.GetBase(), req.GetDraft(),
	)
	if err != nil {
		log.Error("CreatePR failed", "repo", repo, "error", err)
		return nil, status.Errorf(codes.Internal, "create PR: %v", err)
	}

	log.Info("PR created", "repo", fmt.Sprintf("%s/%s", org, repo), "number", number, "url", url)

	return &repodepotv1.CreatePRResponse{Number: number, Url: url, Title: req.GetTitle()}, nil
}

// ListPRs lists pull requests for a repository.
func (s *RepodepotService) ListPRs(ctx context.Context, req *repodepotv1.ListPRsRequest) (*repodepotv1.ListPRsResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	prs, err := s.ghClient(org, repo).ListPRs(ctx, req.GetState())
	if err != nil {
		logger.Get(ctx).Error("ListPRs failed", "repo", repo, "error", err)
		return nil, status.Errorf(codes.Internal, "list PRs: %v", err)
	}

	out := make([]*repodepotv1.PRInfo, len(prs))
	for i, p := range prs {
		out[i] = toProtoPRInfo(p)
	}

	return &repodepotv1.ListPRsResponse{PullRequests: out}, nil
}

// GetPR returns full details for a single pull request.
func (s *RepodepotService) GetPR(ctx context.Context, req *repodepotv1.GetPRRequest) (*repodepotv1.GetPRResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	pr, err := s.ghClient(org, repo).GetPR(ctx, req.GetNumber())
	if err != nil {
		logger.Get(ctx).Error("GetPR failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "get PR: %v", err)
	}

	return &repodepotv1.GetPRResponse{PullRequest: toProtoPRInfo(*pr)}, nil
}

// MergePR merges a pull request.
func (s *RepodepotService) MergePR(ctx context.Context, req *repodepotv1.MergePRRequest) (*repodepotv1.MergePRResponse, error) {
	log := logger.Get(ctx)

	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	if err := s.approve(ctx,
		fmt.Sprintf("Approve: Merge PR #%d?", req.GetNumber()),
		fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, req.GetNumber()),
	); err != nil {
		return nil, err
	}

	if err := s.ghClient(org, repo).MergePR(ctx, req.GetNumber(), req.GetMethod()); err != nil {
		log.Error("MergePR failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "merge PR: %v", err)
	}

	log.Info("PR merged", "repo", fmt.Sprintf("%s/%s", org, repo), "number", req.GetNumber())

	return &repodepotv1.MergePRResponse{}, nil
}

// RequestReview requests a review from one or more users.
func (s *RepodepotService) RequestReview(ctx context.Context, req *repodepotv1.RequestReviewRequest) (*repodepotv1.RequestReviewResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	if len(req.GetReviewers()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one reviewer is required")
	}

	if err := s.ghClient(org, repo).RequestReview(ctx, req.GetNumber(), req.GetReviewers()); err != nil {
		logger.Get(ctx).Error("RequestReview failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "request review: %v", err)
	}

	return &repodepotv1.RequestReviewResponse{}, nil
}

// ListPRComments returns all comments (issue and review) on a pull request.
func (s *RepodepotService) ListPRComments(ctx context.Context, req *repodepotv1.ListPRCommentsRequest) (*repodepotv1.ListPRCommentsResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	client := s.ghClient(org, repo)

	issueComments, err := client.ListIssueComments(ctx, req.GetNumber())
	if err != nil {
		logger.Get(ctx).Error("ListIssueComments failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "list issue comments: %v", err)
	}

	reviewComments, err := client.ListReviewComments(ctx, req.GetNumber())
	if err != nil {
		logger.Get(ctx).Error("ListReviewComments failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "list review comments: %v", err)
	}

	all := append(issueComments, reviewComments...)
	out := make([]*repodepotv1.PRComment, len(all))

	for i, c := range all {
		out[i] = &repodepotv1.PRComment{
			Id:          c.ID,
			Author:      c.Author,
			Body:        c.Body,
			CreatedAt:   c.CreatedAt,
			CommentType: c.CommentType,
			InReplyTo:   c.InReplyTo,
			Path:        c.Path,
			Line:        c.Line,
			Url:         c.URL,
		}
	}

	return &repodepotv1.ListPRCommentsResponse{Comments: out}, nil
}

// CommentOnPR posts a comment on a pull request.
func (s *RepodepotService) CommentOnPR(ctx context.Context, req *repodepotv1.CommentOnPRRequest) (*repodepotv1.CommentOnPRResponse, error) {
	log := logger.Get(ctx)

	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	if req.GetBody() == "" {
		return nil, status.Error(codes.InvalidArgument, "body is required")
	}

	if err := s.approve(ctx,
		fmt.Sprintf("Approve: Post comment on PR #%d?", req.GetNumber()),
		fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, req.GetNumber()),
	); err != nil {
		return nil, err
	}

	client := s.ghClient(org, repo)

	var (
		commentID int64
		url       string
	)

	if req.GetInReplyTo() != 0 {
		commentID, url, err = client.ReplyToReviewComment(ctx, req.GetNumber(), req.GetInReplyTo(), req.GetBody())
	} else {
		commentID, url, err = client.CreateIssueComment(ctx, req.GetNumber(), req.GetBody())
	}

	if err != nil {
		log.Error("CommentOnPR failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "comment on PR: %v", err)
	}

	log.Info("PR comment posted", "repo", fmt.Sprintf("%s/%s", org, repo), "number", req.GetNumber(), "comment_id", commentID)

	return &repodepotv1.CommentOnPRResponse{CommentId: commentID, Url: url}, nil
}

// ListPRChecks returns CI check statuses for a pull request.
func (s *RepodepotService) ListPRChecks(ctx context.Context, req *repodepotv1.ListPRChecksRequest) (*repodepotv1.ListPRChecksResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "number is required")
	}

	checks, err := s.ghClient(org, repo).ListChecks(ctx, req.GetNumber())
	if err != nil {
		logger.Get(ctx).Error("ListPRChecks failed", "repo", repo, "number", req.GetNumber(), "error", err)
		return nil, status.Errorf(codes.Internal, "list PR checks: %v", err)
	}

	out := make([]*repodepotv1.CheckInfo, len(checks))
	for i, c := range checks {
		out[i] = &repodepotv1.CheckInfo{
			RunId:      c.RunID,
			Name:       c.Name,
			Status:     c.Status,
			Conclusion: c.Conclusion,
			Url:        c.URL,
		}
	}

	return &repodepotv1.ListPRChecksResponse{Checks: out}, nil
}

// GetCheckLogs returns the failed-step logs for a workflow run.
func (s *RepodepotService) GetCheckLogs(ctx context.Context, req *repodepotv1.GetCheckLogsRequest) (*repodepotv1.GetCheckLogsResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	if req.GetRunId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}

	logs, err := s.ghClient(org, repo).GetRunLogs(ctx, req.GetRunId())
	if err != nil {
		logger.Get(ctx).Error("GetCheckLogs failed", "repo", repo, "run_id", req.GetRunId(), "error", err)
		return nil, status.Errorf(codes.Internal, "get check logs: %v", err)
	}

	return &repodepotv1.GetCheckLogsResponse{Logs: logs}, nil
}

// GetPRTemplate fetches the pull request description template from a GitHub repo.
func (s *RepodepotService) GetPRTemplate(ctx context.Context, req *repodepotv1.GetPRTemplateRequest) (*repodepotv1.GetPRTemplateResponse, error) {
	org, err := s.resolveOrg(req.GetOrg())
	if err != nil {
		return nil, err
	}

	repo := req.GetRepo()
	if repo == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	tmpl, err := s.ghClient(org, repo).GetPRTemplate(ctx)
	if err != nil {
		logger.Get(ctx).Error("GetPRTemplate failed", "repo", repo, "error", err)
		return nil, status.Errorf(codes.Internal, "get PR template: %v", err)
	}

	return &repodepotv1.GetPRTemplateResponse{Template: tmpl}, nil
}

// --- Graphite handlers ---

// gtClient returns a gt.Runner for the named project's workspace.
func (s *RepodepotService) gtClient(name string) *gt.Runner {
	return gt.New(s.depot().WorkspacePath(name))
}

// GtSync syncs the stack with the remote and restacks as needed.
func (s *RepodepotService) GtSync(ctx context.Context, req *repodepotv1.GtSyncRequest) (*repodepotv1.GtSyncResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.gtClient(name).Sync(ctx); err != nil {
		logger.Get(ctx).Error("GtSync failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "gt sync: %v", err)
	}

	return &repodepotv1.GtSyncResponse{}, nil
}

// GtCreate creates a new stacked branch in the project workspace.
func (s *RepodepotService) GtCreate(ctx context.Context, req *repodepotv1.GtCreateRequest) (*repodepotv1.GtCreateResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if req.GetBranch() == "" {
		return nil, status.Error(codes.InvalidArgument, "branch is required")
	}

	if req.GetMessage() == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	if err := s.gtClient(name).Create(ctx, req.GetBranch(), req.GetMessage()); err != nil {
		logger.Get(ctx).Error("GtCreate failed", "name", name, "branch", req.GetBranch(), "error", err)
		return nil, status.Errorf(codes.Internal, "gt create: %v", err)
	}

	return &repodepotv1.GtCreateResponse{}, nil
}

// GtSubmit submits the stack (or current branch) to GitHub as PRs.
func (s *RepodepotService) GtSubmit(ctx context.Context, req *repodepotv1.GtSubmitRequest) (*repodepotv1.GtSubmitResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	scope := "current branch"
	if req.GetStack() {
		scope = "stack"
	}

	if err := s.approve(ctx,
		"Approve: Submit Graphite PRs?",
		fmt.Sprintf("Project: %s (%s)", name, scope),
	); err != nil {
		return nil, err
	}

	out, err := s.gtClient(name).Submit(ctx, req.GetStack(), req.GetDraft(), req.GetTitle(), req.GetBody())
	if err != nil {
		log.Error("GtSubmit failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "gt submit: %v", err)
	}

	log.Info("gt submit complete", "name", name, "stack", req.GetStack())

	return &repodepotv1.GtSubmitResponse{Output: out}, nil
}

// GtUp moves up the stack by n branches.
func (s *RepodepotService) GtUp(ctx context.Context, req *repodepotv1.GtUpRequest) (*repodepotv1.GtUpResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	branch, err := s.gtClient(name).Up(ctx, req.GetSteps())
	if err != nil {
		logger.Get(ctx).Error("GtUp failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "gt up: %v", err)
	}

	return &repodepotv1.GtUpResponse{Branch: branch}, nil
}

// GtDown moves down the stack by n branches.
func (s *RepodepotService) GtDown(ctx context.Context, req *repodepotv1.GtDownRequest) (*repodepotv1.GtDownResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	branch, err := s.gtClient(name).Down(ctx, req.GetSteps())
	if err != nil {
		logger.Get(ctx).Error("GtDown failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "gt down: %v", err)
	}

	return &repodepotv1.GtDownResponse{Branch: branch}, nil
}

// GtLog returns the current stack as text.
func (s *RepodepotService) GtLog(ctx context.Context, req *repodepotv1.GtLogRequest) (*repodepotv1.GtLogResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	out, err := s.gtClient(name).Log(ctx)
	if err != nil {
		logger.Get(ctx).Error("GtLog failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "gt log: %v", err)
	}

	return &repodepotv1.GtLogResponse{Output: out}, nil
}

// resolveProject returns the project from the request, falling back to the configured default.
func (s *RepodepotService) resolveProject(requestProject string) (string, error) {
	if requestProject != "" {
		return requestProject, nil
	}

	if s.cfg.GCloud.DefaultProject == "" {
		return "", status.Error(codes.InvalidArgument, "project is required (no gcloud.default_project configured)")
	}

	return s.cfg.GCloud.DefaultProject, nil
}

// --- GCB handlers ---

// ListBuilds returns GCB builds matching the given filters.
func (s *RepodepotService) ListBuilds(ctx context.Context, req *repodepotv1.ListBuildsRequest) (*repodepotv1.ListBuildsResponse, error) {
	project, err := s.resolveProject(req.GetProject())
	if err != nil {
		return nil, err
	}

	builds, err := gcloud.New(project).ListBuilds(ctx,
		req.GetCommitSha(), req.GetPrNumber(), req.GetBranch(), req.GetLimit(),
	)
	if err != nil {
		logger.Get(ctx).Error("ListBuilds failed", "project", project, "error", err)
		return nil, status.Errorf(codes.Internal, "list builds: %v", err)
	}

	out := make([]*repodepotv1.BuildInfo, len(builds))
	for i, b := range builds {
		out[i] = &repodepotv1.BuildInfo{
			Id:         b.ID,
			Status:     b.Status,
			CreateTime: b.CreateTime,
			FinishTime: b.FinishTime,
			LogUrl:     b.LogURL,
			CommitSha:  b.CommitSHA,
			Branch:     b.Branch,
			PrNumber:   b.PRNumber,
		}
	}

	return &repodepotv1.ListBuildsResponse{Builds: out}, nil
}

// GetBuildLogs returns the log output for a specific GCB build.
func (s *RepodepotService) GetBuildLogs(ctx context.Context, req *repodepotv1.GetBuildLogsRequest) (*repodepotv1.GetBuildLogsResponse, error) {
	project, err := s.resolveProject(req.GetProject())
	if err != nil {
		return nil, err
	}

	if req.GetBuildId() == "" {
		return nil, status.Error(codes.InvalidArgument, "build_id is required")
	}

	logs, err := gcloud.New(project).GetBuildLogs(ctx, req.GetBuildId())
	if err != nil {
		logger.Get(ctx).Error("GetBuildLogs failed", "project", project, "build_id", req.GetBuildId(), "error", err)
		return nil, status.Errorf(codes.Internal, "get build logs: %v", err)
	}

	return &repodepotv1.GetBuildLogsResponse{Logs: logs}, nil
}

// --- Sync handlers ---

// FetchRepo refreshes the bare cache for a remote URL.
func (s *RepodepotService) FetchRepo(ctx context.Context, req *repodepotv1.FetchRepoRequest) (*repodepotv1.FetchRepoResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	remoteURL := req.GetRemoteUrl()
	if remoteURL == "" {
		return nil, status.Error(codes.InvalidArgument, "remote_url is required")
	}

	if err := s.depot().FetchRepo(ctx, remoteURL); err != nil {
		log.Error("FetchRepo failed", "name", name, "remote", remoteURL, "error", err)
		return nil, status.Errorf(codes.Internal, "fetch repo: %v", err)
	}

	log.Info("repo fetched", "name", name, "remote", remoteURL)

	return &repodepotv1.FetchRepoResponse{}, nil
}

// PullRepo merges upstream changes for a subtree into the project workspace.
func (s *RepodepotService) PullRepo(ctx context.Context, req *repodepotv1.PullRepoRequest) (*repodepotv1.PullRepoResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	remoteURL := req.GetRemoteUrl()
	if remoteURL == "" {
		return nil, status.Error(codes.InvalidArgument, "remote_url is required")
	}

	if err := s.depot().PullRepo(ctx, name, remoteURL, req.GetUseGraphite()); err != nil {
		log.Error("PullRepo failed", "name", name, "remote", remoteURL, "error", err)
		return nil, status.Errorf(codes.Internal, "pull repo: %v", err)
	}

	log.Info("repo pulled", "name", name, "remote", remoteURL)

	return &repodepotv1.PullRepoResponse{}, nil
}

// SyncProject fetches or pulls all subtrees in a project.
func (s *RepodepotService) SyncProject(ctx context.Context, req *repodepotv1.SyncProjectRequest) (*repodepotv1.SyncProjectResponse, error) {
	log := logger.Get(ctx)

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.depot().SyncProject(ctx, name, req.GetFetchOnly(), req.GetUseGraphite()); err != nil {
		log.Error("SyncProject failed", "name", name, "error", err)
		return nil, status.Errorf(codes.Internal, "sync project: %v", err)
	}

	log.Info("project synced", "name", name, "fetch_only", req.GetFetchOnly())

	return &repodepotv1.SyncProjectResponse{}, nil
}

// toProtoPRInfo converts a gh.PRInfo to the proto type.
func toProtoPRInfo(p gh.PRInfo) *repodepotv1.PRInfo {
	return &repodepotv1.PRInfo{
		Number:     p.Number,
		Title:      p.Title,
		State:      p.State,
		Url:        p.URL,
		Author:     p.Author,
		HeadBranch: p.HeadBranch,
		BaseBranch: p.BaseBranch,
		CreatedAt:  p.CreatedAt,
		Body:       p.Body,
	}
}

// repoName extracts the repository name from a remote URL.
//
//	"https://github.com/foo/bar.git" → "bar"
//	"git@github.com:foo/bar"         → "bar"
func repoName(remoteURL string) string {
	base := filepath.Base(strings.TrimSuffix(remoteURL, ".git"))
	if base == "" || base == "." {
		return "repo"
	}

	return base
}
