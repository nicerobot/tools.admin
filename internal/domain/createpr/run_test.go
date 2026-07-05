package createpr

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/gitcmd"
	"github.com/nicerobot/tools.admin/internal/repo"
)

type fakeGit struct {
	errOn      string
	branch     repo.Branch
	stagePath  gitcmd.StagePath
	commitMsg  gitcmd.CommitMessage
	pushBranch repo.Branch
	calls      []string
	prArgs     []string
	staged     bool
	prExists   bool
}

func (f *fakeGit) maybe(m string) error {
	f.calls = append(f.calls, m)
	if f.errOn == m {
		return errors.New(m + " failed")
	}
	return nil
}

func (f *fakeGit) ConfigureBotIdentity() error        { return f.maybe("config") }
func (f *fakeGit) CheckoutBranch(b repo.Branch) error { f.branch = b; return f.maybe("checkout") }

func (f *fakeGit) StageDirectory(p gitcmd.StagePath) error { f.stagePath = p; return f.maybe("stage") }

func (f *fakeGit) HasStagedChanges() (bool, error) {
	f.calls = append(f.calls, "staged")
	if f.errOn == "staged" {
		return false, errors.New("staged failed")
	}
	return f.staged, nil
}

func (f *fakeGit) Commit(m gitcmd.CommitMessage) error { f.commitMsg = m; return f.maybe("commit") }
func (f *fakeGit) ForcePush(b repo.Branch) error       { f.pushBranch = b; return f.maybe("push") }

func (f *fakeGit) PrExists(repo.Branch) (bool, error) {
	f.calls = append(f.calls, "prexists")
	if f.errOn == "prexists" {
		return false, errors.New("prexists failed")
	}
	return f.prExists, nil
}

func (f *fakeGit) CreatePR(t gitcmd.PRTitle, b gitcmd.PRBody, head repo.Branch, base repo.Base) error {
	f.prArgs = []string{string(t), string(b), string(head), string(base)}
	return f.maybe("createpr")
}

func runWith(t *testing.T, g *fakeGit, branch repo.Branch, base repo.Base) (Result, error) {
	t.Helper()
	orig := deps
	t.Cleanup(func() { deps = orig })
	deps = func() dependencies { return dependencies{git: g} }
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return Run(context.Background(), logger, Config{SettingsPath: ".github", Branch: branch, Base: base})
}

func TestFullFlowWithChanges(t *testing.T) {
	g := &fakeGit{staged: true, prExists: false}
	res, err := runWith(t, g, "settings-sync/snapshot", "main")
	require.NoError(t, err)
	assert.Equal(t, Result{
		Branch: "settings-sync/snapshot", HasChanges: true, IsCommitted: true, IsPushed: true, IsPRCreated: true,
	}, res)
	assert.Equal(t, gitcmd.StagePath(".github/repos"), g.stagePath)
	assert.Equal(t, repo.Branch("settings-sync/snapshot"), g.pushBranch)
	assert.Equal(t, gitcmd.CommitMessage("chore: snapshot live repo settings"), g.commitMsg)
	assert.Equal(t, []string{
		"chore: snapshot live repo settings",
		"Auto-generated snapshot of current GitHub repo settings vs org/account defaults.",
		"settings-sync/snapshot",
		"main",
	}, g.prArgs)
}

func TestNoChangesExitsEarly(t *testing.T) {
	g := &fakeGit{staged: false}
	res, err := runWith(t, g, "settings-sync/snapshot", "main")
	require.NoError(t, err)
	assert.Equal(t, Result{Branch: "settings-sync/snapshot"}, res)
	assert.NotContains(t, g.calls, "commit")
}

func TestExistingPRSkipsCreation(t *testing.T) {
	g := &fakeGit{staged: true, prExists: true}
	res, err := runWith(t, g, "settings-sync/snapshot", "main")
	require.NoError(t, err)
	assert.True(t, res.IsPRAlreadyOpen)
	assert.False(t, res.IsPRCreated)
	assert.Nil(t, g.prArgs)
}

func TestCustomBranchAndBase(t *testing.T) {
	g := &fakeGit{staged: true, prExists: false}
	_, err := runWith(t, g, "custom/branch", "develop")
	require.NoError(t, err)
	assert.Equal(t, repo.Branch("custom/branch"), g.branch)
	assert.Equal(t, repo.Branch("custom/branch"), g.pushBranch)
	assert.Equal(t, "custom/branch", g.prArgs[2])
	assert.Equal(t, "develop", g.prArgs[3])
}

func TestErrorPaths(t *testing.T) {
	for _, stage := range []string{"config", "checkout", "stage", "staged", "commit", "push", "prexists", "createpr"} {
		g := &fakeGit{staged: true, prExists: false, errOn: stage}
		_, err := runWith(t, g, "b", "main")
		require.Error(t, err, "stage %s should error", stage)
	}
}

// TestOSDeps exercises the production wiring builds a git service.
func TestOSDeps(t *testing.T) {
	d := osDeps()
	assert.NotNil(t, d.git)
}
