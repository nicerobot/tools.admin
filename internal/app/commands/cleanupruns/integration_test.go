//go:build integration

package cleanupruns

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/nicerobot/tools.admin/internal/app"
)

// TestIntegration_CleanupRunsDryRun assembles the real cli.Command and runs
// `cleanup-runs --dry-run` end-to-end against an httptest GitHub API, exercising
// the full app → domain → github wiring with no real network and no deletions.
func TestIntegration_CleanupRunsDryRun(t *testing.T) {
	want, must := assert.New(t), require.New(t)

	var deleted int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/widget/actions/runs":
			_, _ = w.Write([]byte(`{"workflow_runs":[` +
				`{"id":1,"workflow_id":7,"name":"CI","status":"completed","created_at":"2024-01-01T00:00:00Z"},` +
				`{"id":2,"workflow_id":7,"name":"CI","status":"completed","created_at":"2024-01-02T00:00:00Z"}` +
				`]}`))
		case r.Method == http.MethodDelete:
			deleted++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("GH_TOKEN", "tok")
	t.Setenv("GITHUB_API_URL", server.URL)

	var stdout bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError}))
	appCmd := &cli.Command{
		Name:     "app",
		Writer:   &stdout,
		Commands: []*cli.Command{Command()},
		Metadata: map[string]any{app.LoggerMetadataKey: logger},
	}

	err := appCmd.Run(context.Background(), []string{
		"app", "cleanup-runs",
		"--owner", "acme", "--repo", "widget",
		"--days", "0", "--keep", "0", "--dry-run",
	})
	must.NoError(err)

	out := stdout.String()
	want.Contains(out, `"dry_run": true`)
	want.Contains(out, `"deleted": 2`)
	want.Contains(out, `"repos_scanned": 1`)
	want.Zero(deleted, "dry-run must not issue any DELETE")
}
