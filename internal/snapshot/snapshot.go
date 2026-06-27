// Package snapshot orchestrates the snapshot command: load org defaults, list
// live repos, diff each to an override file, and reconcile stale files — but
// only after verifying every stale candidate is truly gone, so a token that
// cannot see all repos can never delete a file it merely failed to list.
package snapshot

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
	"github.com/nicerobot/tools.admin/internal/overrides"
	"github.com/nicerobot/tools.admin/internal/settings"
)

// GitHub is the API surface snapshot needs.
type GitHub interface {
	GetAccountType(owner domain.Owner) (domain.AccountType, error)
	ListRepos(owner domain.Owner) ([]githubmodel.Repository, error)
	RepoExists(owner domain.Owner, name domain.RepoName) (bool, error)
}

// Deps are snapshot's injected collaborators.
type Deps struct {
	GitHub       GitHub
	Load         func(path domain.SettingsPath) (settings.OrgSettings, error)
	Write        func(file overrides.File, reposDir overrides.ReposDir) (overrides.OutFile, error)
	ListExisting func(reposDir overrides.ReposDir) ([]string, error)
	Remove       overrides.RemoveFunc
	Out          io.Writer
}

// Run executes the snapshot command.
func Run(d Deps, owner domain.Owner, settingsPath domain.SettingsPath) error {
	s, err := d.Load(settingsPath)
	if err != nil {
		return err
	}
	at, err := d.GitHub.GetAccountType(owner)
	if err != nil {
		return err
	}
	repos, err := d.GitHub.ListRepos(owner)
	if err != nil {
		return err
	}
	reposDir := overrides.ReposDir(filepath.Join(string(settingsPath), "repos"))
	files, live := computeFiles(repos, s.Repository, owner, commentSource(at))
	gone, err := verifyStale(d, owner, reposDir, live)
	if err != nil {
		return err
	}
	return apply(d, reposDir, files, gone)
}

func commentSource(at domain.AccountType) domain.CommentSource {
	if at == domain.AccountTypeOrganization {
		return domain.CommentSourceOrg
	}
	return domain.CommentSourceAccount
}

func computeFiles(
	repos []githubmodel.Repository,
	defaults settings.RepositoryDefaults,
	owner domain.Owner,
	source domain.CommentSource,
) ([]overrides.File, map[string]bool) {
	files := make([]overrides.File, 0, len(repos))
	live := make(map[string]bool, len(repos))
	for _, r := range repos {
		live[r.Name] = true
		files = append(files, overrides.Compute(r, defaults, owner, source))
	}
	return files, live
}

func verifyStale(d Deps, owner domain.Owner, reposDir overrides.ReposDir, live map[string]bool) ([]string, error) {
	existing, err := d.ListExisting(reposDir)
	if err != nil {
		return nil, err
	}
	gone := make([]string, 0)
	for _, name := range staleNames(existing, live) {
		exists, err := d.GitHub.RepoExists(owner, domain.RepoName(name))
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, adminerr.ErrStaleRepoExists.With(nil, "repo", string(owner)+"/"+name)
		}
		gone = append(gone, name)
	}
	return gone, nil
}

func staleNames(existing []string, live map[string]bool) []string {
	stale := make([]string, 0)
	for _, name := range existing {
		if !live[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	return stale
}

func apply(d Deps, reposDir overrides.ReposDir, files []overrides.File, gone []string) error {
	if err := writeAll(d, reposDir, files); err != nil {
		return err
	}
	return removeGone(d, reposDir, gone)
}

func writeAll(d Deps, reposDir overrides.ReposDir, files []overrides.File) error {
	for _, f := range files {
		out, err := d.Write(f, reposDir)
		if err != nil {
			return err
		}
		fmt.Fprintf(d.Out, "  wrote %s\n", out)
	}
	return nil
}

func removeGone(d Deps, reposDir overrides.ReposDir, gone []string) error {
	for _, name := range gone {
		path := filepath.Join(string(reposDir), name+".yml")
		fmt.Fprintf(d.Out, "  removing %s (repo no longer exists)\n", path)
		if err := d.Remove(path); err != nil {
			return adminerr.ErrRemoveFile.With(err, "file", path)
		}
	}
	return nil
}
