// Package cleanupruns orchestrates the cleanup-runs command: resolve the target
// owner/repo (explicit or from GITHUB_REPOSITORY), then per repo list completed
// runs older than the cutoff, group by workflow, keep the newest N, and delete
// (or, in dry-run, report) the rest.
package cleanupruns

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/nicerobot/tools.admin/internal/adminerr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/githubmodel"
)

const dateLayout = "2006-01-02"

// GitHub is the API surface cleanup-runs needs.
type GitHub interface {
	ListRepos(owner domain.Owner) ([]githubmodel.Repository, error)
	ListWorkflowRuns(owner domain.Owner, repo domain.RepoName, before domain.CreatedBefore) ([]githubmodel.WorkflowRun, error)
	DeleteWorkflowRun(owner domain.Owner, repo domain.RepoName, id domain.RunID) error
}

// Options are the cleanup-runs parameters. An empty Owner triggers
// GITHUB_REPOSITORY auto-detection; an empty Repo scans all repos under owner.
type Options struct {
	Owner  domain.Owner
	Repo   domain.RepoName
	Days   domain.Days
	Keep   domain.KeepCount
	DryRun domain.DryRun
}

// Deps are cleanup-runs' injected collaborators.
type Deps struct {
	GitHub GitHub
	Env    func(key string) string
	Now    func() time.Time
	Out    io.Writer
}

// Run executes the cleanup-runs command.
func Run(d Deps, o Options) error {
	owner, repo, err := resolveTarget(o.Owner, o.Repo, d.Env)
	if err != nil {
		return err
	}
	names, err := targetRepos(d, owner, repo)
	if err != nil {
		return err
	}
	cutoff := cutoffDate(d.Now, o.Days)
	deleted, kept, err := pruneRepos(d, owner, names, cutoff, o.Keep, o.DryRun)
	if err != nil {
		return err
	}
	printSummary(d.Out, len(names), deleted, kept, o.DryRun)
	return nil
}

func resolveTarget(owner domain.Owner, repo domain.RepoName, env func(string) string) (domain.Owner, domain.RepoName, error) {
	if owner != "" {
		return owner, repo, nil
	}
	o, n, ok := strings.Cut(env("GITHUB_REPOSITORY"), "/")
	if !ok || o == "" || n == "" {
		return "", "", adminerr.ErrNoTarget.With(nil)
	}
	return domain.Owner(o), domain.RepoName(n), nil
}

func cutoffDate(now func() time.Time, days domain.Days) domain.CreatedBefore {
	return domain.CreatedBefore(now().UTC().AddDate(0, 0, -int(days)).Format(dateLayout))
}

func targetRepos(d Deps, owner domain.Owner, repo domain.RepoName) ([]domain.RepoName, error) {
	if repo != "" {
		return []domain.RepoName{repo}, nil
	}
	repos, err := d.GitHub.ListRepos(owner)
	if err != nil {
		return nil, err
	}
	names := make([]domain.RepoName, 0, len(repos))
	for _, r := range repos {
		names = append(names, domain.RepoName(r.Name))
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names, nil
}

func pruneRepos(
	d Deps,
	owner domain.Owner,
	names []domain.RepoName,
	cutoff domain.CreatedBefore,
	keep domain.KeepCount,
	dry domain.DryRun,
) (int, int, error) {
	totalDeleted, totalKept := 0, 0
	for _, name := range names {
		deleted, kept, err := pruneRepo(d, owner, name, cutoff, keep, dry)
		if err != nil {
			return 0, 0, err
		}
		totalDeleted += deleted
		totalKept += kept
	}
	return totalDeleted, totalKept, nil
}

func pruneRepo(
	d Deps,
	owner domain.Owner,
	name domain.RepoName,
	cutoff domain.CreatedBefore,
	keep domain.KeepCount,
	dry domain.DryRun,
) (int, int, error) {
	runs, err := d.GitHub.ListWorkflowRuns(owner, name, cutoff)
	if err != nil {
		return 0, 0, err
	}
	if len(runs) == 0 {
		return 0, 0, nil
	}
	toDelete, kept := partition(runs, keep)
	if len(toDelete) == 0 {
		return 0, 0, nil
	}
	fmt.Fprintf(d.Out, "%s/%s: deleting %d, keeping %d\n", owner, name, len(toDelete), kept)
	if err := deleteRuns(d, owner, name, toDelete, dry); err != nil {
		return 0, 0, err
	}
	return len(toDelete), kept, nil
}

func partition(runs []githubmodel.WorkflowRun, keep domain.KeepCount) ([]githubmodel.WorkflowRun, int) {
	var toDelete []githubmodel.WorkflowRun
	kept := 0
	for _, group := range groupByWorkflow(runs) {
		sortNewestFirst(group)
		k := min(len(group), int(keep))
		kept += k
		toDelete = append(toDelete, group[k:]...)
	}
	return toDelete, kept
}

func groupByWorkflow(runs []githubmodel.WorkflowRun) [][]githubmodel.WorkflowRun {
	index := map[int64]int{}
	var groups [][]githubmodel.WorkflowRun
	for _, r := range runs {
		i, ok := index[r.WorkflowID]
		if !ok {
			i = len(groups)
			index[r.WorkflowID] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], r)
	}
	return groups
}

func sortNewestFirst(runs []githubmodel.WorkflowRun) {
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].CreatedAt > runs[j].CreatedAt })
}

func deleteRuns(d Deps, owner domain.Owner, name domain.RepoName, runs []githubmodel.WorkflowRun, dry domain.DryRun) error {
	for _, r := range runs {
		label := fmt.Sprintf("  run %d (%s, %s)", r.ID, r.Name, r.CreatedAt)
		if bool(dry) {
			fmt.Fprintf(d.Out, "  [dry-run] would delete %s\n", label)
			continue
		}
		if err := d.GitHub.DeleteWorkflowRun(owner, name, domain.RunID(r.ID)); err != nil {
			return err
		}
		fmt.Fprintf(d.Out, "  deleted %s\n", label)
	}
	return nil
}

func printSummary(out io.Writer, scanned, deleted, kept int, dry domain.DryRun) {
	action := "deleted"
	if bool(dry) {
		action = "would delete"
	}
	fmt.Fprintf(out, "\nSummary: %d repos scanned, %d runs %s, %d runs kept\n", scanned, deleted, action, kept)
}
