// Package gh provides utilities for shelling out to the GitHub CLI (gh).
package gh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/amp-labs/amp-common/cmd"
)

// Runner executes gh commands against a specific GitHub repository.
type Runner struct {
	repo string // "org/repo"
}

// New creates a Runner targeting the given org/repo slug.
func New(repo string) *Runner {
	return &Runner{repo: repo}
}

// run executes a gh command, streaming stdout and stderr to process output.
func (r *Runner) run(ctx context.Context, args ...string) error {
	code, err := cmd.New(ctx, "gh", args...).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return fmt.Errorf("gh %s: exited with code %d", strings.Join(args, " "), code)
	}

	return nil
}

// outputBytes executes a gh command and returns raw stdout bytes.
// stderr is still streamed to process output.
func (r *Runner) outputBytes(ctx context.Context, args ...string) ([]byte, error) {
	var buf []byte

	code, err := cmd.New(ctx, "gh", args...).
		SetStdoutObserver(func(b []byte) { buf = append(buf, b...) }).
		SetStderr(os.Stderr).
		Run()
	if err != nil {
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}

	if code != 0 {
		return nil, fmt.Errorf("gh %s: exited with code %d", strings.Join(args, " "), code)
	}

	return buf, nil
}

// outputString executes a gh command and returns trimmed stdout as a string.
func (r *Runner) outputString(ctx context.Context, args ...string) (string, error) {
	b, err := r.outputBytes(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}

// --- Internal JSON types ---

type ghAuthor struct {
	Login string `json:"login"`
}

type ghPR struct {
	Number      int64    `json:"number"`
	Title       string   `json:"title"`
	State       string   `json:"state"`
	URL         string   `json:"url"`
	Author      ghAuthor `json:"author"`
	HeadRefName string   `json:"headRefName"`
	BaseRefName string   `json:"baseRefName"`
	CreatedAt   string   `json:"createdAt"`
	Body        string   `json:"body"`
}

type ghIssueComment struct {
	ID        int64    `json:"id"`
	User      ghAuthor `json:"user"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at"`
	HTMLURL   string   `json:"html_url"`
}

type ghReviewComment struct {
	ID          int64    `json:"id"`
	User        ghAuthor `json:"user"`
	Body        string   `json:"body"`
	CreatedAt   string   `json:"created_at"`
	HTMLURL     string   `json:"html_url"`
	InReplyToID int64    `json:"in_reply_to_id"`
	Path        string   `json:"path"`
	Line        int32    `json:"line"`
}

type ghCheck struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Conclusion string `json:"conclusion"`
	Link       string `json:"link"`
}

type ghAPIComment struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
}

// --- Domain types returned to callers ---

// PRInfo holds summary information about a pull request.
type PRInfo struct {
	Number     int64
	Title      string
	State      string
	URL        string
	Author     string
	HeadBranch string
	BaseBranch string
	CreatedAt  string
	Body       string
}

// PRComment holds a single comment (issue or review) on a pull request.
type PRComment struct {
	ID          int64
	Author      string
	Body        string
	CreatedAt   string
	CommentType string // "issue" or "review"
	InReplyTo   int64
	Path        string
	Line        int32
	URL         string
}

// CheckInfo holds status information about a single CI check.
type CheckInfo struct {
	RunID      int64
	Name       string
	Status     string
	Conclusion string
	URL        string
}

// prJSON is the fields we request from gh pr list/view.
const prJSON = "number,title,state,url,author,headRefName,baseRefName,createdAt,body"

// checkJSON is the fields we request from gh pr checks.
const checkJSON = "name,state,conclusion,link"

// runIDFromURL parses a workflow run ID from a GitHub Actions URL.
// e.g. https://github.com/owner/repo/actions/runs/12345678/job/987654321 → 12345678
var runIDRe = regexp.MustCompile(`/runs/(\d+)`)

func parseRunID(link string) int64 {
	m := runIDRe.FindStringSubmatch(link)
	if len(m) < 2 {
		return 0
	}

	id, _ := strconv.ParseInt(m[1], 10, 64)

	return id
}

// prNumberFromURL parses a PR number from a GitHub PR URL.
// e.g. https://github.com/owner/repo/pull/42 → 42
var prNumberRe = regexp.MustCompile(`/pull/(\d+)`)

func parsePRNumber(url string) int64 {
	m := prNumberRe.FindStringSubmatch(url)
	if len(m) < 2 {
		return 0
	}

	n, _ := strconv.ParseInt(m[1], 10, 64)

	return n
}

func toPRInfo(p ghPR) PRInfo {
	return PRInfo{
		Number:     p.Number,
		Title:      p.Title,
		State:      strings.ToLower(p.State),
		URL:        p.URL,
		Author:     p.Author.Login,
		HeadBranch: p.HeadRefName,
		BaseBranch: p.BaseRefName,
		CreatedAt:  p.CreatedAt,
		Body:       p.Body,
	}
}

// --- Public methods ---

// CreatePR opens a new pull request and returns its number and URL.
func (r *Runner) CreatePR(ctx context.Context, title, body, head, base string, draft bool) (int64, string, error) {
	if base == "" {
		base = "main"
	}

	args := []string{
		"pr", "create",
		"--repo", r.repo,
		"--title", title,
		"--body", body,
		"--head", head,
		"--base", base,
	}

	if draft {
		args = append(args, "--draft")
	}

	out, err := r.outputString(ctx, args...)
	if err != nil {
		return 0, "", err
	}

	// gh pr create outputs the PR URL on success.
	number := parsePRNumber(out)

	return number, out, nil
}

// ListPRs returns pull requests for the repository filtered by state.
// state may be "open", "closed", "merged", or "all" (default: "open").
func (r *Runner) ListPRs(ctx context.Context, state string) ([]PRInfo, error) {
	if state == "" {
		state = "open"
	}

	b, err := r.outputBytes(ctx, "pr", "list",
		"--repo", r.repo,
		"--state", state,
		"--json", prJSON,
	)
	if err != nil {
		return nil, err
	}

	var raw []ghPR
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse pr list: %w", err)
	}

	prs := make([]PRInfo, len(raw))
	for i, p := range raw {
		prs[i] = toPRInfo(p)
	}

	return prs, nil
}

// GetPR returns details for a single pull request.
func (r *Runner) GetPR(ctx context.Context, number int64) (*PRInfo, error) {
	b, err := r.outputBytes(ctx, "pr", "view",
		strconv.FormatInt(number, 10),
		"--repo", r.repo,
		"--json", prJSON,
	)
	if err != nil {
		return nil, err
	}

	var raw ghPR
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse pr view: %w", err)
	}

	info := toPRInfo(raw)

	return &info, nil
}

// MergePR merges a pull request. method may be "merge", "squash", or "rebase".
func (r *Runner) MergePR(ctx context.Context, number int64, method string) error {
	if method == "" {
		method = "merge"
	}

	return r.run(ctx,
		"pr", "merge",
		strconv.FormatInt(number, 10),
		"--repo", r.repo,
		"--"+method,
	)
}

// RequestReview requests reviews from the given GitHub usernames.
func (r *Runner) RequestReview(ctx context.Context, number int64, reviewers []string) error {
	return r.run(ctx,
		"pr", "edit",
		strconv.FormatInt(number, 10),
		"--repo", r.repo,
		"--add-reviewer", strings.Join(reviewers, ","),
	)
}

// ListIssueComments returns top-level (issue) comments on a pull request.
func (r *Runner) ListIssueComments(ctx context.Context, number int64) ([]PRComment, error) {
	b, err := r.outputBytes(ctx,
		"api", fmt.Sprintf("repos/%s/issues/%d/comments", r.repo, number),
	)
	if err != nil {
		return nil, err
	}

	var raw []ghIssueComment
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse issue comments: %w", err)
	}

	out := make([]PRComment, len(raw))
	for i, c := range raw {
		out[i] = PRComment{
			ID:          c.ID,
			Author:      c.User.Login,
			Body:        c.Body,
			CreatedAt:   c.CreatedAt,
			CommentType: "issue",
			URL:         c.HTMLURL,
		}
	}

	return out, nil
}

// ListReviewComments returns inline review comments on a pull request.
func (r *Runner) ListReviewComments(ctx context.Context, number int64) ([]PRComment, error) {
	b, err := r.outputBytes(ctx,
		"api", fmt.Sprintf("repos/%s/pulls/%d/comments", r.repo, number),
	)
	if err != nil {
		return nil, err
	}

	var raw []ghReviewComment
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse review comments: %w", err)
	}

	out := make([]PRComment, len(raw))
	for i, c := range raw {
		out[i] = PRComment{
			ID:          c.ID,
			Author:      c.User.Login,
			Body:        c.Body,
			CreatedAt:   c.CreatedAt,
			CommentType: "review",
			InReplyTo:   c.InReplyToID,
			Path:        c.Path,
			Line:        c.Line,
			URL:         c.HTMLURL,
		}
	}

	return out, nil
}

// CreateIssueComment posts a top-level comment on a pull request.
func (r *Runner) CreateIssueComment(ctx context.Context, number int64, body string) (int64, string, error) {
	b, err := r.outputBytes(ctx,
		"api", fmt.Sprintf("repos/%s/issues/%d/comments", r.repo, number),
		"--method", "POST",
		"--field", "body="+body,
	)
	if err != nil {
		return 0, "", err
	}

	var resp ghAPIComment
	if err := json.Unmarshal(b, &resp); err != nil {
		return 0, "", fmt.Errorf("parse comment response: %w", err)
	}

	return resp.ID, resp.HTMLURL, nil
}

// ReplyToReviewComment posts a reply in an existing review comment thread.
func (r *Runner) ReplyToReviewComment(ctx context.Context, prNumber, inReplyTo int64, body string) (int64, string, error) {
	b, err := r.outputBytes(ctx,
		"api", fmt.Sprintf("repos/%s/pulls/%d/comments", r.repo, prNumber),
		"--method", "POST",
		"--field", "body="+body,
		"--field", fmt.Sprintf("in_reply_to=%d", inReplyTo),
	)
	if err != nil {
		return 0, "", err
	}

	var resp ghAPIComment
	if err := json.Unmarshal(b, &resp); err != nil {
		return 0, "", fmt.Errorf("parse reply response: %w", err)
	}

	return resp.ID, resp.HTMLURL, nil
}

// ListChecks returns CI check statuses for a pull request.
func (r *Runner) ListChecks(ctx context.Context, number int64) ([]CheckInfo, error) {
	b, err := r.outputBytes(ctx,
		"pr", "checks",
		strconv.FormatInt(number, 10),
		"--repo", r.repo,
		"--json", checkJSON,
	)
	if err != nil {
		return nil, err
	}

	var raw []ghCheck
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse pr checks: %w", err)
	}

	out := make([]CheckInfo, len(raw))
	for i, c := range raw {
		out[i] = CheckInfo{
			RunID:      parseRunID(c.Link),
			Name:       c.Name,
			Status:     strings.ToLower(c.State),
			Conclusion: strings.ToLower(c.Conclusion),
			URL:        c.Link,
		}
	}

	return out, nil
}

// GetPRTemplate fetches the pull request description template from the repository.
// Looks for .github/pull_request_template.md (case-insensitive on GitHub's side).
func (r *Runner) GetPRTemplate(ctx context.Context) (string, error) {
	b, err := r.outputBytes(ctx,
		"api", fmt.Sprintf("repos/%s/contents/.github/pull_request_template.md", r.repo),
	)
	if err != nil {
		return "", err
	}

	var resp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.Unmarshal(b, &resp); err != nil {
		return "", fmt.Errorf("parse template response: %w", err)
	}

	if resp.Encoding != "base64" {
		return "", fmt.Errorf("unexpected encoding %q from GitHub", resp.Encoding)
	}

	// GitHub inserts newlines into the base64 content — strip them before decoding.
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(resp.Content, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("decode template content: %w", err)
	}

	return string(decoded), nil
}

// GetRunLogs returns the failed-step logs for a workflow run.
func (r *Runner) GetRunLogs(ctx context.Context, runID int64) (string, error) {
	return r.outputString(ctx,
		"run", "view",
		strconv.FormatInt(runID, 10),
		"--repo", r.repo,
		"--log-failed",
	)
}
