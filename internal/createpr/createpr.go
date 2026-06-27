// Package createpr orchestrates the create-pr command: configure the bot
// identity, checkout the snapshot branch, stage the repos directory, and — only
// when something is staged — commit, force-push, and open a PR if none is open.
package createpr

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/gitsvc"
)

const (
	commitMessage = "chore: snapshot live repo settings"
	prTitle       = "chore: snapshot live repo settings"
	prBody        = "Auto-generated snapshot of current GitHub repo settings vs org/account defaults."
)

// Git is the git/gh surface create-pr needs.
type Git interface {
	ConfigureBotIdentity() error
	CheckoutBranch(branch domain.Branch) error
	StageDirectory(path gitsvc.StagePath) error
	HasStagedChanges() (bool, error)
	Commit(message gitsvc.CommitMessage) error
	ForcePush(branch domain.Branch) error
	PrExists(branch domain.Branch) (bool, error)
	CreatePR(title gitsvc.PRTitle, body gitsvc.PRBody, head domain.Branch, base domain.Base) error
}

// Deps are create-pr's injected collaborators.
type Deps struct {
	Git Git
	Out io.Writer
}

// Run executes the create-pr command.
func Run(d Deps, settingsPath domain.SettingsPath, branch domain.Branch, base domain.Base) error {
	if err := prepare(d, settingsPath, branch); err != nil {
		return err
	}
	staged, err := d.Git.HasStagedChanges()
	if err != nil {
		return err
	}
	if !staged {
		fmt.Fprintln(d.Out, "No changes to commit.")
		return nil
	}
	return commitAndPR(d, branch, base)
}

func prepare(d Deps, settingsPath domain.SettingsPath, branch domain.Branch) error {
	if err := d.Git.ConfigureBotIdentity(); err != nil {
		return err
	}
	if err := d.Git.CheckoutBranch(branch); err != nil {
		return err
	}
	return d.Git.StageDirectory(gitsvc.StagePath(filepath.Join(string(settingsPath), "repos")))
}

func commitAndPR(d Deps, branch domain.Branch, base domain.Base) error {
	if err := d.Git.Commit(commitMessage); err != nil {
		return err
	}
	if err := d.Git.ForcePush(branch); err != nil {
		return err
	}
	exists, err := d.Git.PrExists(branch)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return d.Git.CreatePR(prTitle, prBody, branch, base)
}
