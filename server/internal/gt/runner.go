// Package gt provides utilities for shelling out to the Graphite CLI (gt).
// Graphite maintains stack metadata inside the git repo, so all commands
// must run from within the project workspace directory.
package gt

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/cmd"
)

// Runner executes gt commands rooted at a specific workspace directory.
type Runner struct {
	dir string
}

// New creates a Runner whose working directory is dir.
func New(dir string) *Runner {
	return &Runner{dir: dir}
}

// run executes a gt command, streaming stdout and stderr to process output.
func (r *Runner) run(ctx context.Context, args ...string) error {
	code, err := cmd.New(ctx, "gt", args...).
		SetDir(r.dir).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return fmt.Errorf("gt %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return fmt.Errorf("gt %s: exited with code %d", strings.Join(args, " "), code)
	}

	return nil
}

// outputString executes a gt command and returns trimmed stdout as a string.
// stderr is still streamed to process output.
func (r *Runner) outputString(ctx context.Context, args ...string) (string, error) {
	var buf []byte

	code, err := cmd.New(ctx, "gt", args...).
		SetDir(r.dir).
		SetStdoutObserver(func(b []byte) { buf = append(buf, b...) }).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return "", fmt.Errorf("gt %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return "", fmt.Errorf("gt %s: exited with code %d", strings.Join(args, " "), code)
	}

	return strings.TrimSpace(string(buf)), nil
}

// Sync syncs the stack with the remote and restacks as needed.
func (r *Runner) Sync(ctx context.Context) error {
	return r.run(ctx, "sync")
}

// Create creates a new branch stacked on the current branch and commits
// any staged changes with the given commit message.
func (r *Runner) Create(ctx context.Context, branch, message string) error {
	return r.run(ctx, "create", branch, "-m", message)
}

// Submit submits the stack (or current branch) to GitHub as PRs.
// Pass stack=true to submit all branches in the stack.
// title and body apply to new PRs; pass empty strings to use commit message defaults.
func (r *Runner) Submit(ctx context.Context, stack, draft bool, title, body string) (string, error) {
	args := []string{"submit", "--no-interactive"}

	if stack {
		args = append(args, "--stack")
	}

	if draft {
		args = append(args, "--draft")
	}

	if title != "" {
		args = append(args, "--title", title)
	}

	if body != "" {
		args = append(args, "--body", body)
	}

	return r.outputString(ctx, args...)
}

// Up moves up the stack by n branches. Returns the name of the branch
// now checked out.
func (r *Runner) Up(ctx context.Context, steps int32) (string, error) {
	if steps <= 0 {
		steps = 1
	}

	args := []string{"up"}
	if steps > 1 {
		args = append(args, fmt.Sprintf("%d", steps))
	}

	if err := r.run(ctx, args...); err != nil {
		return "", err
	}

	return r.currentBranch(ctx)
}

// Down moves down the stack by n branches. Returns the name of the branch
// now checked out.
func (r *Runner) Down(ctx context.Context, steps int32) (string, error) {
	if steps <= 0 {
		steps = 1
	}

	args := []string{"down"}
	if steps > 1 {
		args = append(args, fmt.Sprintf("%d", steps))
	}

	if err := r.run(ctx, args...); err != nil {
		return "", err
	}

	return r.currentBranch(ctx)
}

// Log returns the current stack as text output from gt log.
func (r *Runner) Log(ctx context.Context) (string, error) {
	return r.outputString(ctx, "log")
}

// currentBranch returns the name of the currently checked-out branch.
func (r *Runner) currentBranch(ctx context.Context) (string, error) {
	var buf []byte

	code, err := cmd.New(ctx, "git", "branch", "--show-current").
		SetDir(r.dir).
		SetStdoutObserver(func(b []byte) { buf = append(buf, b...) }).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w", err)
	}

	if code != 0 {
		return "", fmt.Errorf("git branch --show-current: exited with code %d", code)
	}

	return strings.TrimSpace(string(buf)), nil
}
