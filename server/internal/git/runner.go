// Package git provides utilities for shelling out to git.
package git

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/cmd"
)

// Runner executes git commands rooted at a specific directory.
type Runner struct {
	dir string
}

// New creates a Runner whose working directory is dir.
func New(dir string) *Runner {
	return &Runner{dir: dir}
}

// Dir returns the working directory.
func (r *Runner) Dir() string {
	return r.dir
}

// run executes a git command, streaming stdout and stderr to the process output.
func (r *Runner) run(ctx context.Context, args ...string) error {
	code, err := cmd.New(ctx, "git", args...).
		SetDir(r.dir).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return fmt.Errorf("git %s: exited with code %d", strings.Join(args, " "), code)
	}

	return nil
}

// output executes a git command and returns its stdout as a trimmed string.
// stderr is still streamed to the process output.
func (r *Runner) output(ctx context.Context, args ...string) (string, error) {
	var out string

	code, err := cmd.New(ctx, "git", args...).
		SetDir(r.dir).
		SetStdoutObserver(func(b []byte) { out = strings.TrimSpace(string(b)) }).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return "", fmt.Errorf("git %s: exited with code %d", strings.Join(args, " "), code)
	}

	return out, nil
}

// InitBare initializes a bare git repository at the runner's directory.
func (r *Runner) InitBare(ctx context.Context) error {
	return r.run(ctx, "init", "--bare")
}

// CloneMirror creates a bare mirror clone of remoteURL at the runner's directory.
// This is used to seed the depot cache for a remote repository.
func (r *Runner) CloneMirror(ctx context.Context, remoteURL string) error {
	args := []string{"clone", "--mirror", remoteURL, r.dir}

	code, err := cmd.New(ctx, "git", args...).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return fmt.Errorf("git clone --mirror %s: %w", remoteURL, err)
	}

	if code != 0 {
		return fmt.Errorf("git clone --mirror %s: exited with code %d", remoteURL, code)
	}

	return nil
}

// Clone clones from src into the runner's directory.
// Use local=true for fast on-disk clones from a local bare cache.
func (r *Runner) Clone(ctx context.Context, src string, local bool) error {
	args := []string{"clone"}
	if local {
		args = append(args, "--local")
	}

	args = append(args, src, r.dir)

	code, err := cmd.New(ctx, "git", args...).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return fmt.Errorf("git clone %s: %w", src, err)
	}

	if code != 0 {
		return fmt.Errorf("git clone %s: exited with code %d", src, code)
	}

	return nil
}

// SubtreeAdd adds a remote as a subtree under prefix at the given branch.
func (r *Runner) SubtreeAdd(ctx context.Context, prefix, src, branch string) error {
	return r.run(ctx, "subtree", "add", "--prefix="+prefix, src, branch, "--squash")
}

// SubtreePush pushes a subtree back to its remote at the given branch.
func (r *Runner) SubtreePush(ctx context.Context, prefix, remote, branch string) error {
	return r.run(ctx, "subtree", "push", "--prefix="+prefix, remote, branch)
}

// CheckoutBranch creates and checks out a new branch, or just checks it out if it exists.
func (r *Runner) CheckoutBranch(ctx context.Context, branch string) error {
	// Try to checkout existing branch first.
	code, err := cmd.New(ctx, "git", "checkout", branch).
		SetDir(r.dir).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err == nil && code == 0 {
		return nil
	}

	// Branch doesn't exist yet — create it.
	return r.run(ctx, "checkout", "-b", branch)
}

// AddAll stages all changes.
func (r *Runner) AddAll(ctx context.Context) error {
	return r.run(ctx, "add", "-A")
}

// Commit creates a commit with the given message. Returns nil if there was
// nothing to commit.
func (r *Runner) Commit(ctx context.Context, message string) error {
	// Check if there's anything to commit.
	out, err := r.output(ctx, "status", "--porcelain")
	if err != nil {
		return err
	}

	if out == "" {
		return nil
	}

	return r.run(ctx, "commit", "-m", message)
}

// Push pushes the current branch to remote.
func (r *Runner) Push(ctx context.Context, remote, branch string) error {
	return r.run(ctx, "push", remote, branch)
}

// Fetch fetches from all remotes.
func (r *Runner) Fetch(ctx context.Context) error {
	return r.run(ctx, "fetch", "--all")
}

// RemoteSetURL sets the URL for a named remote.
func (r *Runner) RemoteSetURL(ctx context.Context, name, url string) error {
	return r.run(ctx, "remote", "set-url", name, url)
}

// RemoteGetURL returns the URL for a named remote.
func (r *Runner) RemoteGetURL(ctx context.Context, name string) (string, error) {
	return r.output(ctx, "remote", "get-url", name)
}

// SubtreePull merges upstream changes into the current branch for a subtree.
func (r *Runner) SubtreePull(ctx context.Context, prefix, src, branch string) error {
	return r.run(ctx, "subtree", "pull", "--prefix="+prefix, src, branch, "--squash")
}

// CurrentBranch returns the name of the currently checked-out branch.
func (r *Runner) CurrentBranch(ctx context.Context) (string, error) {
	return r.output(ctx, "branch", "--show-current")
}

// RebaseOnto rebases the current branch onto the named branch.
func (r *Runner) RebaseOnto(ctx context.Context, branch string) error {
	return r.run(ctx, "rebase", branch)
}

// RevParse resolves a ref (e.g. "HEAD", a branch, or a SHA) to a full commit SHA.
func (r *Runner) RevParse(ctx context.Context, ref string) (string, error) {
	sha, err := r.output(ctx, "rev-parse", "--verify", ref)
	if err != nil {
		return "", fmt.Errorf("ref %q not found: %w", ref, err)
	}

	return sha, nil
}
