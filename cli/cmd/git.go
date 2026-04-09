package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// currentProjectName walks up from the working directory to find a git repo,
// reads its origin remote URL, and derives the project name from it.
// Returns the project name and the repo root path.
func currentProjectName() (name, repoRoot string, err error) {
	repoRoot, err = gitRepoRoot()
	if err != nil {
		return "", "", fmt.Errorf("not inside a git repository: %w", err)
	}

	originURL, err := gitRemoteGetURL(repoRoot, "origin")
	if err != nil {
		return "", "", fmt.Errorf("no origin remote: %w", err)
	}

	name = projectNameFromBarePath(originURL)
	if name == "" {
		return "", "", fmt.Errorf("could not derive project name from origin %q", originURL)
	}

	return name, repoRoot, nil
}

// gitRepoRoot returns the root of the git repository containing the working directory.
func gitRepoRoot() (string, error) {
	out, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return out, nil
}

// gitRemoteGetURL returns the URL of a named remote.
func gitRemoteGetURL(dir, remote string) (string, error) {
	return runGit(dir, "remote", "get-url", remote)
}

// gitRemoteSetURL updates the URL of a named remote.
func gitRemoteSetURL(dir, remote, url string) error {
	_, err := runGit(dir, "remote", "set-url", remote, url)

	return err
}

// runGit runs a git command in dir (empty = CWD) and returns trimmed stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// projectNameFromBarePath extracts the project name from a bare repo path.
// e.g. "/depot/projects/my-project.git" → "my-project"
func projectNameFromBarePath(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ".git")

	if name == "" || name == "." {
		return ""
	}

	return name
}

// findGitDir walks up from the given directory looking for a .git directory.
// Returns the path containing .git, or an error if none is found.
func findGitDir(start string) (string, error) {
	dir := start

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git directory found above %s", start)
		}

		dir = parent
	}
}
