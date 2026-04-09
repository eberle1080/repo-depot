// Package depot manages the on-disk layout of the repo-depot bare clone cache.
//
// Layout under depotPath:
//
//	depotPath/
//	├── projects/   bare repos for project containers (local authority)
//	└── repos/      bare mirror caches of real remote repositories
package depot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amp-labs/amp-common/logger"
	"github.com/eberle1080/repo-depot/server/internal/git"
	"github.com/eberle1080/repo-depot/server/internal/gt"
)

const (
	projectsDir = "projects"
	reposDir    = "repos"

	initialCommitMessage = "initial commit"
	saveCommitMessage    = "repo-depot: save"
)

// Depot manages the depot directory layout and orchestrates git operations.
type Depot struct {
	root           string
	workspacesPath string
}

// New creates a Depot rooted at depotPath, with workspaces under workspacesPath.
func New(depotPath, workspacesPath string) *Depot {
	return &Depot{root: depotPath, workspacesPath: workspacesPath}
}

// WorkspacePath returns the host-side path for a project workspace.
func (d *Depot) WorkspacePath(name string) string {
	return filepath.Join(d.workspacesPath, name)
}

// projectBarePath returns the path of a project's bare repo in the depot.
func (d *Depot) projectBarePath(name string) string {
	return filepath.Join(d.root, projectsDir, name+".git")
}

// repoBarePath returns the path of a remote repo's bare cache in the depot.
// The URL structure is preserved as a relative path so different hosts/orgs
// don't collide.
// e.g. "https://github.com/foo/bar" → "<depot>/repos/github.com/foo/bar.git"
func (d *Depot) repoBarePath(remoteURL string) string {
	rel := urlToPath(remoteURL)
	return filepath.Join(d.root, reposDir, rel)
}

// CreateProject initializes a new project: creates a bare repo under
// depot/projects/, clones it to workspacesPath/<name>, creates an initial
// commit, and pushes back so the bare is non-empty.
func (d *Depot) CreateProject(ctx context.Context, name string) error {
	log := logger.Get(ctx)

	barePath := d.projectBarePath(name)

	if _, err := os.Stat(barePath); err == nil {
		return fmt.Errorf("project %q already exists in depot", name)
	}

	if err := os.MkdirAll(barePath, 0o755); err != nil {
		return fmt.Errorf("create project bare dir: %w", err)
	}

	bare := git.New(barePath)
	if err := bare.InitBare(ctx); err != nil {
		return err
	}

	log.Debug("project bare repo initialised", "path", barePath)

	workspacePath := d.WorkspacePath(name)
	if err := os.MkdirAll(filepath.Dir(workspacePath), 0o755); err != nil {
		return fmt.Errorf("create workspace parent: %w", err)
	}

	ws := git.New(workspacePath)
	if err := ws.Clone(ctx, barePath, true); err != nil {
		return err
	}

	// A bare repo with no commits can't be branched from. Write a .gitkeep so
	// the initial commit gives us a real HEAD to work from.
	keepFile := filepath.Join(workspacePath, ".gitkeep")
	if err := os.WriteFile(keepFile, []byte(""), 0o644); err != nil {
		return fmt.Errorf("write .gitkeep: %w", err)
	}

	if err := ws.AddAll(ctx); err != nil {
		return err
	}

	if err := ws.Commit(ctx, initialCommitMessage); err != nil {
		return err
	}

	if err := ws.Push(ctx, "origin", "main"); err != nil {
		return err
	}

	log.Debug("project workspace ready", "path", workspacePath)

	return nil
}

// CloneRepo ensures the remote URL has a bare cache in the depot, then adds it
// as a subtree inside the named project's workspace at the given prefix.
func (d *Depot) CloneRepo(ctx context.Context, name, remoteURL, prefix string, readOnly bool) (string, error) {
	log := logger.Get(ctx)

	barePath, err := d.ensureRepoCache(ctx, remoteURL)
	if err != nil {
		return "", err
	}

	workspacePath := d.WorkspacePath(name)

	branch := "main"
	if !readOnly {
		branch = name
		log.Debug("using project branch for rw repo", "branch", branch)
	}

	ws := git.New(workspacePath)
	if err := ws.SubtreeAdd(ctx, prefix, barePath, branch); err != nil {
		return "", fmt.Errorf("subtree add %q at %q: %w", remoteURL, prefix, err)
	}

	if err := d.writeRepoMeta(workspacePath, prefix, remoteURL, readOnly); err != nil {
		return "", err
	}

	return prefix, nil
}

// SaveProject pushes each rw subtree back to its bare cache, commits the
// project state, and pushes the project repo to its bare via origin.
func (d *Depot) SaveProject(ctx context.Context, name string) error {
	workspacePath := d.WorkspacePath(name)

	metas, err := d.readAllRepoMeta(workspacePath)
	if err != nil {
		return fmt.Errorf("read repo metadata: %w", err)
	}

	ws := git.New(workspacePath)

	for _, m := range metas {
		if m.ReadOnly {
			continue
		}

		barePath := d.repoBarePath(m.RemoteURL)

		if err := ws.SubtreePush(ctx, m.Prefix, barePath, name); err != nil {
			return fmt.Errorf("subtree push %q: %w", m.Prefix, err)
		}
	}

	if err := ws.AddAll(ctx); err != nil {
		return err
	}

	if err := ws.Commit(ctx, saveCommitMessage); err != nil {
		return err
	}

	// Push via origin — stays correct through renames.
	return ws.Push(ctx, "origin", "main")
}

// ListProjects returns the names of all projects in the depot.
func (d *Depot) ListProjects() ([]string, error) {
	dir := filepath.Join(d.root, projectsDir)

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read projects dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		name := e.Name()
		if filepath.Ext(name) == ".git" {
			names = append(names, name[:len(name)-4])
		}
	}

	return names, nil
}

// RenameProject renames the bare repo in the depot and returns the new bare path.
// Updating any worktree remotes is the caller's responsibility.
func (d *Depot) RenameProject(ctx context.Context, oldName, newName string) (string, error) {
	oldPath := d.projectBarePath(oldName)
	newPath := d.projectBarePath(newName)

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return "", fmt.Errorf("project %q not found in depot", oldName)
	}

	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("project %q already exists in depot", newName)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("rename bare repo: %w", err)
	}

	logger.Get(ctx).Info("project bare repo renamed", "old", oldName, "new", newName)

	return newPath, nil
}

// CheckoutProject re-creates the workspace for an existing project by cloning
// its bare repo. Use after DeleteProject or on a fresh machine.
func (d *Depot) CheckoutProject(ctx context.Context, name string) error {
	barePath := d.projectBarePath(name)

	if _, err := os.Stat(barePath); os.IsNotExist(err) {
		return fmt.Errorf("project %q not found in depot", name)
	}

	workspacePath := d.WorkspacePath(name)

	if _, err := os.Stat(workspacePath); err == nil {
		return fmt.Errorf("workspace for project %q already exists at %s", name, workspacePath)
	}

	if err := os.MkdirAll(filepath.Dir(workspacePath), 0o755); err != nil {
		return fmt.Errorf("create workspace parent: %w", err)
	}

	ws := git.New(workspacePath)
	if err := ws.Clone(ctx, barePath, true); err != nil {
		return err
	}

	logger.Get(ctx).Info("project workspace checked out", "name", name, "path", workspacePath)

	return nil
}

// DeleteProject archives the workspace to a timestamped zip and removes it.
// The bare repo in the depot is preserved.
func (d *Depot) DeleteProject(ctx context.Context, name string) (string, error) {
	workspacePath := d.WorkspacePath(name)
	logger.Get(ctx).Info("archiving workspace", "name", name, "path", workspacePath)

	return d.ArchiveAndDelete(workspacePath)
}

// FetchRepo refreshes the bare cache for remoteURL without touching any workspace.
func (d *Depot) FetchRepo(ctx context.Context, remoteURL string) error {
	_, err := d.ensureRepoCache(ctx, remoteURL)
	return err
}

// PullRepo merges upstream changes for the subtree identified by remoteURL into
// the named project's workspace.
func (d *Depot) PullRepo(ctx context.Context, name, remoteURL string, useGraphite bool) error {
	barePath, err := d.ensureRepoCache(ctx, remoteURL)
	if err != nil {
		return err
	}

	workspacePath := d.WorkspacePath(name)

	metas, err := d.readAllRepoMeta(workspacePath)
	if err != nil {
		return fmt.Errorf("read repo metadata: %w", err)
	}

	var meta *repoMeta
	for i := range metas {
		if metas[i].RemoteURL == remoteURL {
			meta = &metas[i]
			break
		}
	}

	if meta == nil {
		return fmt.Errorf("repo %q not found in project %q", remoteURL, name)
	}

	branch := "main"
	if !meta.ReadOnly {
		branch = name
	}

	ws := git.New(workspacePath)

	currentBranch, err := ws.CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	if currentBranch == "main" {
		return ws.SubtreePull(ctx, meta.Prefix, barePath, branch)
	}

	// On a feature branch: switch to main, pull, switch back, rebase.
	if err := ws.CheckoutBranch(ctx, "main"); err != nil {
		return fmt.Errorf("checkout main: %w", err)
	}

	if err := ws.SubtreePull(ctx, meta.Prefix, barePath, branch); err != nil {
		return fmt.Errorf("subtree pull %q: %w", meta.Prefix, err)
	}

	if err := ws.CheckoutBranch(ctx, currentBranch); err != nil {
		return fmt.Errorf("checkout %q: %w", currentBranch, err)
	}

	if useGraphite {
		if err := gt.New(workspacePath).Sync(ctx); err != nil {
			return fmt.Errorf("gt sync: %w", err)
		}
	} else {
		if err := ws.RebaseOnto(ctx, "main"); err != nil {
			return fmt.Errorf("rebase onto main: %w", err)
		}
	}

	return nil
}

// SyncProject fetches or pulls all subtrees in a project.
func (d *Depot) SyncProject(ctx context.Context, name string, fetchOnly, useGraphite bool) error {
	workspacePath := d.WorkspacePath(name)

	metas, err := d.readAllRepoMeta(workspacePath)
	if err != nil {
		return fmt.Errorf("read repo metadata: %w", err)
	}

	for _, m := range metas {
		if fetchOnly {
			if err := d.FetchRepo(ctx, m.RemoteURL); err != nil {
				return fmt.Errorf("fetch %q: %w", m.Prefix, err)
			}
		} else {
			if err := d.PullRepo(ctx, name, m.RemoteURL, useGraphite); err != nil {
				return fmt.Errorf("pull %q: %w", m.Prefix, err)
			}
		}
	}

	return nil
}

// ensureRepoCache returns the path to the bare cache for remoteURL, creating
// or updating it as needed.
func (d *Depot) ensureRepoCache(ctx context.Context, remoteURL string) (string, error) {
	log := logger.Get(ctx)

	barePath := d.repoBarePath(remoteURL)

	if _, err := os.Stat(barePath); os.IsNotExist(err) {
		log.Info("seeding bare cache", "remote", remoteURL, "path", barePath)

		if err := os.MkdirAll(filepath.Dir(barePath), 0o755); err != nil {
			return "", fmt.Errorf("create cache dir: %w", err)
		}

		r := git.New(barePath)
		if err := r.CloneMirror(ctx, remoteURL); err != nil {
			return "", fmt.Errorf("mirror clone %q: %w", remoteURL, err)
		}
	} else {
		log.Info("refreshing bare cache", "remote", remoteURL)

		r := git.New(barePath)
		if err := r.Fetch(ctx); err != nil {
			return "", fmt.Errorf("fetch cache %q: %w", remoteURL, err)
		}
	}

	return barePath, nil
}

// urlToPath converts a remote URL to a relative filesystem path suitable for
// use under the repos/ directory.
//
//	https://github.com/foo/bar     → github.com/foo/bar.git
//	https://github.com/foo/bar.git → github.com/foo/bar.git
//	git@github.com:foo/bar.git     → github.com/foo/bar.git
func urlToPath(remoteURL string) string {
	s := remoteURL

	// Strip scheme.
	for _, prefix := range []string{"https://", "http://", "git://"} {
		s = strings.TrimPrefix(s, prefix)
	}

	// Normalise SCP-style git@host:path → host/path.
	if idx := strings.Index(s, "@"); idx != -1 {
		s = s[idx+1:]
		s = strings.Replace(s, ":", "/", 1)
	}

	// Ensure .git suffix.
	if !strings.HasSuffix(s, ".git") {
		s += ".git"
	}

	return filepath.FromSlash(s)
}
