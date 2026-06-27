// Package cli builds the tools.admin urfave/cli/v3 command tree and wires each
// subcommand to its orchestration package through an injected Env. Keeping the
// wiring here (not in main) makes every Action unit-testable with fake seams,
// while cmd/tools.admin/main.go stays a thin, untested composition root.
package cli

import (
	"context"
	"io"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/nicerobot/tools.admin/internal/cleanupruns"
	"github.com/nicerobot/tools.admin/internal/createpr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubapi"
	"github.com/nicerobot/tools.admin/internal/gitsvc"
	"github.com/nicerobot/tools.admin/internal/overrides"
	"github.com/nicerobot/tools.admin/internal/settings"
	"github.com/nicerobot/tools.admin/internal/snapshot"
)

const defaultSettingsPath = ".github"

// Env bundles every injectable seam the commands need. Production wiring
// (cmd/tools.admin) provides real implementations; tests provide fakes.
type Env struct {
	Out      io.Writer
	Doer     githubapi.Doer
	BaseURL  string
	Getenv   func(key string) string
	Now      func() time.Time
	GitRun   gitsvc.RunFunc
	ReadFile settings.ReadFileFunc
	Mkdir    overrides.MkdirAllFunc
	WriteOut overrides.WriteFileFunc
	Glob     overrides.GlobFunc
	Remove   overrides.RemoveFunc
}

// NewCommand builds the root tools.admin command from env.
func NewCommand(env Env) *cli.Command {
	return &cli.Command{
		Name:                  "tools.admin",
		Usage:                 "GitHub admin automation tools",
		Writer:                env.Out,
		ErrWriter:             env.Out,
		HideHelpCommand:       true,
		EnableShellCompletion: false,
		Commands: []*cli.Command{
			env.snapshotCommand(),
			env.createPRCommand(),
			env.cleanupRunsCommand(),
		},
	}
}

func (env Env) snapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "Snapshot live repo settings",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "owner", Usage: "GitHub user or organization", Required: true},
			&cli.StringFlag{Name: "settings-path", Usage: "Path to settings directory", Value: defaultSettingsPath},
		},
		Action: env.snapshotAction,
	}
}

func (env Env) createPRCommand() *cli.Command {
	return &cli.Command{
		Name:  "create-pr",
		Usage: "Create PR from snapshot",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "settings-path", Usage: "Path to settings directory", Value: defaultSettingsPath},
			&cli.StringFlag{Name: "branch", Usage: "Branch name", Value: "safe-settings/snapshot"},
			&cli.StringFlag{Name: "base", Usage: "Base branch", Value: "main"},
		},
		Action: env.createPRAction,
	}
}

func (env Env) cleanupRunsCommand() *cli.Command {
	return &cli.Command{
		Name:  "cleanup-runs",
		Usage: "Delete old workflow runs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "owner", Usage: "GitHub user or organization (default: current repo from GITHUB_REPOSITORY)"},
			&cli.StringFlag{Name: "repo", Usage: "Single repo (omit for all repos)"},
			&cli.IntFlag{Name: "days", Usage: "Delete runs older than N days", Value: 30},
			&cli.IntFlag{Name: "keep", Usage: "Keep at least N runs per workflow", Value: 5},
			&cli.BoolFlag{Name: "dry-run", Usage: "Print what would be deleted without deleting"},
		},
		Action: env.cleanupRunsAction,
	}
}

func (env Env) snapshotAction(_ context.Context, cmd *cli.Command) error {
	client, err := env.newClient()
	if err != nil {
		return err
	}
	deps := snapshot.Deps{
		GitHub:       client,
		Load:         func(p domain.SettingsPath) (settings.OrgSettings, error) { return settings.Load(env.ReadFile, p) },
		Write:        env.writeOverride,
		ListExisting: env.listExisting,
		Remove:       env.Remove,
		Out:          env.Out,
	}
	owner := domain.Owner(cmd.String("owner"))
	return snapshot.Run(deps, owner, domain.SettingsPath(cmd.String("settings-path")))
}

func (env Env) createPRAction(_ context.Context, cmd *cli.Command) error {
	deps := createpr.Deps{Git: gitsvc.New(env.GitRun), Out: env.Out}
	return createpr.Run(
		deps,
		domain.SettingsPath(cmd.String("settings-path")),
		domain.Branch(cmd.String("branch")),
		domain.Base(cmd.String("base")),
	)
}

func (env Env) cleanupRunsAction(_ context.Context, cmd *cli.Command) error {
	client, err := env.newClient()
	if err != nil {
		return err
	}
	deps := cleanupruns.Deps{GitHub: client, Env: env.Getenv, Now: env.Now, Out: env.Out}
	opts := cleanupruns.Options{
		Owner:  domain.Owner(cmd.String("owner")),
		Repo:   domain.RepoName(cmd.String("repo")),
		Days:   domain.Days(cmd.Int("days")),
		Keep:   domain.KeepCount(cmd.Int("keep")),
		DryRun: domain.DryRun(cmd.Bool("dry-run")),
	}
	return cleanupruns.Run(deps, opts)
}

func (env Env) newClient() (githubapi.Client, error) {
	return githubapi.New(env.Doer, env.BaseURL, env.Getenv)
}

func (env Env) writeOverride(file overrides.File, reposDir overrides.ReposDir) (overrides.OutFile, error) {
	return overrides.Write(file, reposDir, env.Mkdir, env.WriteOut)
}

func (env Env) listExisting(reposDir overrides.ReposDir) ([]string, error) {
	return overrides.ListExisting(reposDir, env.Glob)
}
