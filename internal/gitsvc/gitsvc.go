// Package gitsvc wraps the git and gh subprocesses behind an injected runner so
// the create-pr orchestration is unit-testable with no real subprocess. The
// command set and arguments mirror the original CLI exactly.
package gitsvc

import (
	"strings"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
)

const (
	botName  = "github-actions[bot]"
	botEmail = "41898282+github-actions[bot]@users.noreply.github.com"
)

// CommitMessage is a git commit message.
type CommitMessage string

// StagePath is a path passed to `git add --all`.
type StagePath string

// PRTitle is a pull-request title.
type PRTitle string

// PRBody is a pull-request body.
type PRBody string

// Result is the outcome of a single command invocation.
type Result struct {
	Stdout   string
	ExitCode int
}

// RunFunc runs a command and returns its result. A non-nil error means the
// command could not be executed at all; a non-zero ExitCode means it ran and
// failed. Injected so tests never spawn a process.
type RunFunc func(args []string) (Result, error)

// Service runs git/gh operations through an injected runner. It is an immutable
// value safe to copy.
type Service struct {
	run RunFunc
}

// New builds a Service from an injected runner.
func New(run RunFunc) Service { return Service{run: run} }

// ConfigureBotIdentity sets the github-actions[bot] git identity.
func (s Service) ConfigureBotIdentity() error {
	if err := s.checked("git", "config", "user.name", botName); err != nil {
		return err
	}
	return s.checked("git", "config", "user.email", botEmail)
}

// CheckoutBranch force-creates and checks out branch.
func (s Service) CheckoutBranch(branch domain.Branch) error {
	return s.checked("git", "checkout", "-B", string(branch))
}

// StageDirectory stages every change under path.
func (s Service) StageDirectory(path StagePath) error {
	return s.checked("git", "add", "--all", string(path))
}

// HasStagedChanges reports whether anything is staged (git diff --cached
// --quiet exits non-zero when there are staged changes).
func (s Service) HasStagedChanges() (bool, error) {
	res, err := s.run([]string{"git", "diff", "--cached", "--quiet"})
	if err != nil {
		return false, adminerr.ErrCommand.With(err, "args", "git diff --cached --quiet")
	}
	return res.ExitCode != 0, nil
}

// Commit records a commit with the given message.
func (s Service) Commit(message CommitMessage) error {
	return s.checked("git", "commit", "-m", string(message))
}

// ForcePush force-pushes branch to origin.
func (s Service) ForcePush(branch domain.Branch) error {
	return s.checked("git", "push", "--force", "origin", string(branch))
}

// PrExists reports whether an open PR already exists for the head branch.
func (s Service) PrExists(branch domain.Branch) (bool, error) {
	res, err := s.run([]string{
		"gh", "pr", "list",
		"--head", string(branch),
		"--state", "open",
		"--json", "number",
		"--jq", ".[0].number",
	})
	if err != nil {
		return false, adminerr.ErrCommand.With(err, "args", "gh pr list")
	}
	return strings.TrimSpace(res.Stdout) != "", nil
}

// CreatePR opens a pull request from head into base.
func (s Service) CreatePR(title PRTitle, body PRBody, head domain.Branch, base domain.Base) error {
	return s.checked(
		"gh", "pr", "create",
		"--title", string(title),
		"--body", string(body),
		"--head", string(head),
		"--base", string(base),
	)
}

func (s Service) checked(args ...string) error {
	res, err := s.run(args)
	if err != nil {
		return adminerr.ErrCommand.With(err, "args", strings.Join(args, " "))
	}
	if res.ExitCode != 0 {
		return adminerr.ErrCommand.With(nil, "args", strings.Join(args, " "), "code", res.ExitCode)
	}
	return nil
}
