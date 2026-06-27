package createpr_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/createpr"
	"github.com/nicerobot/tools.admin/internal/domain"
	"github.com/nicerobot/tools.admin/internal/gitsvc"
)

type fakeGit struct {
	staged   bool
	prExists bool
	errOn    string
	calls    []string

	branch     domain.Branch
	stagePath  gitsvc.StagePath
	commitMsg  gitsvc.CommitMessage
	pushBranch domain.Branch
	prArgs     []string
}

func (f *fakeGit) maybe(m string) error {
	f.calls = append(f.calls, m)
	if f.errOn == m {
		return errors.New(m + " failed")
	}
	return nil
}

func (f *fakeGit) ConfigureBotIdentity() error          { return f.maybe("config") }
func (f *fakeGit) CheckoutBranch(b domain.Branch) error { f.branch = b; return f.maybe("checkout") }

func (f *fakeGit) StageDirectory(p gitsvc.StagePath) error { f.stagePath = p; return f.maybe("stage") }

func (f *fakeGit) HasStagedChanges() (bool, error) {
	f.calls = append(f.calls, "staged")
	if f.errOn == "staged" {
		return false, errors.New("staged failed")
	}
	return f.staged, nil
}

func (f *fakeGit) Commit(m gitsvc.CommitMessage) error { f.commitMsg = m; return f.maybe("commit") }
func (f *fakeGit) ForcePush(b domain.Branch) error     { f.pushBranch = b; return f.maybe("push") }

func (f *fakeGit) PrExists(domain.Branch) (bool, error) {
	f.calls = append(f.calls, "prexists")
	if f.errOn == "prexists" {
		return false, errors.New("prexists failed")
	}
	return f.prExists, nil
}

func (f *fakeGit) CreatePR(t gitsvc.PRTitle, b gitsvc.PRBody, head domain.Branch, base domain.Base) error {
	f.prArgs = []string{string(t), string(b), string(head), string(base)}
	return f.maybe("createpr")
}

func run(g *fakeGit, branch domain.Branch, base domain.Base) (string, error) {
	var out bytes.Buffer
	err := createpr.Run(createpr.Deps{Git: g, Out: &out}, ".github", branch, base)
	return out.String(), err
}

func TestFullFlowWithChanges(t *testing.T) {
	g := &fakeGit{staged: true, prExists: false}
	out, err := run(g, "safe-settings/snapshot", "main")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Equal(t, gitsvc.StagePath(".github/repos"), g.stagePath)
	assert.Equal(t, domain.Branch("safe-settings/snapshot"), g.pushBranch)
	assert.Equal(t, gitsvc.CommitMessage("chore: snapshot live repo settings"), g.commitMsg)
	assert.Equal(t, []string{
		"chore: snapshot live repo settings",
		"Auto-generated snapshot of current GitHub repo settings vs org/account defaults.",
		"safe-settings/snapshot",
		"main",
	}, g.prArgs)
}

func TestNoChangesExitsEarly(t *testing.T) {
	g := &fakeGit{staged: false}
	out, err := run(g, "safe-settings/snapshot", "main")
	require.NoError(t, err)
	assert.Equal(t, "No changes to commit.\n", out)
	assert.NotContains(t, g.calls, "commit")
}

func TestExistingPRSkipsCreation(t *testing.T) {
	g := &fakeGit{staged: true, prExists: true}
	_, err := run(g, "safe-settings/snapshot", "main")
	require.NoError(t, err)
	assert.Nil(t, g.prArgs)
}

func TestCustomBranchAndBase(t *testing.T) {
	g := &fakeGit{staged: true, prExists: false}
	_, err := run(g, "custom/branch", "develop")
	require.NoError(t, err)
	assert.Equal(t, domain.Branch("custom/branch"), g.branch)
	assert.Equal(t, domain.Branch("custom/branch"), g.pushBranch)
	assert.Equal(t, "custom/branch", g.prArgs[2])
	assert.Equal(t, "develop", g.prArgs[3])
}

func TestErrorPaths(t *testing.T) {
	for _, stage := range []string{"config", "checkout", "stage", "staged", "commit", "push", "prexists", "createpr"} {
		g := &fakeGit{staged: true, prExists: false, errOn: stage}
		_, err := run(g, "b", "main")
		require.Error(t, err, "stage %s should error", stage)
	}
}
