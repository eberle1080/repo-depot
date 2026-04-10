// Package gcloud provides utilities for shelling out to the gcloud CLI.
package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/cmd"
)

// Runner executes gcloud commands against a specific GCP project.
type Runner struct {
	project string
}

// New creates a Runner targeting the given GCP project.
func New(project string) *Runner {
	return &Runner{project: project}
}

// outputBytes executes a gcloud command and returns raw stdout bytes.
// stderr is still streamed to process output.
func (r *Runner) outputBytes(ctx context.Context, args ...string) ([]byte, error) {
	var buf []byte

	code, err := cmd.New(ctx, "gcloud", args...).
		SetStdoutObserver(func(b []byte) { buf = append(buf, b...) }).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return nil, fmt.Errorf("gcloud %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return nil, fmt.Errorf("gcloud %s: exited with code %d", strings.Join(args, " "), code)
	}

	return buf, nil
}

func (r *Runner) outputString(ctx context.Context, args ...string) (string, error) {
	b, err := r.outputBytes(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}

// --- Internal JSON types ---

type gcbBuild struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	CreateTime    string            `json:"createTime"`
	FinishTime    string            `json:"finishTime"`
	LogURL        string            `json:"logUrl"`
	Substitutions map[string]string `json:"substitutions"`
}

// --- Domain types returned to callers ---

// BuildInfo holds summary information about a GCB build.
type BuildInfo struct {
	ID         string
	Status     string
	CreateTime string
	FinishTime string
	LogURL     string
	CommitSHA  string
	Branch     string
	PRNumber   string
}

// ListBuilds returns GCB builds matching the given filters.
// At least one of commitSHA, prNumber, or branch should be non-empty.
// limit caps the number of results (default 10).
func (r *Runner) ListBuilds(ctx context.Context, commitSHA, prNumber, branch string, limit int32) ([]BuildInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	var filters []string

	if commitSHA != "" {
		filters = append(filters, fmt.Sprintf(`substitutions.COMMIT_SHA="%s"`, commitSHA))
	}

	if prNumber != "" {
		filters = append(filters, fmt.Sprintf(`substitutions._PR_NUMBER="%s"`, prNumber))
	}

	if branch != "" {
		filters = append(filters, fmt.Sprintf(`substitutions.BRANCH_NAME="%s"`, branch))
	}

	args := []string{
		"builds", "list",
		"--project", r.project,
		"--format", "json",
		"--limit", fmt.Sprintf("%d", limit),
	}

	if len(filters) > 0 {
		args = append(args, "--filter", strings.Join(filters, " AND "))
	}

	b, err := r.outputBytes(ctx, args...)
	if err != nil {
		return nil, err
	}

	var raw []gcbBuild
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse builds list: %w", err)
	}

	out := make([]BuildInfo, len(raw))
	for i, build := range raw {
		out[i] = BuildInfo{
			ID:         build.ID,
			Status:     build.Status,
			CreateTime: build.CreateTime,
			FinishTime: build.FinishTime,
			LogURL:     build.LogURL,
			CommitSHA:  build.Substitutions["COMMIT_SHA"],
			Branch:     build.Substitutions["BRANCH_NAME"],
			PRNumber:   build.Substitutions["_PR_NUMBER"],
		}
	}

	return out, nil
}

// GetBuildLogs returns the log output for a specific GCB build.
func (r *Runner) GetBuildLogs(ctx context.Context, buildID string) (string, error) {
	return r.outputString(ctx, "builds", "log", buildID, "--project", r.project)
}
