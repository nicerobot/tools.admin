package snapshot

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/nicerobot/tools.admin/internal/app"
	domain "github.com/nicerobot/tools.admin/internal/domain/snapshot"
)

func TestSnapshotCommand(t *testing.T) {
	tests := []struct {
		wantCfg      domain.Config
		name         string
		wantContains string
		args         []string
		wantErr      bool
	}{
		{
			name:         "binds owner and settings-path",
			args:         []string{"app", "snapshot", "--owner", "myorg", "--settings-path", ".gh"},
			wantCfg:      domain.Config{Owner: "myorg", SettingsPath: ".gh"},
			wantContains: `"owner": "myorg"`,
		},
		{
			name:         "settings-path defaults",
			args:         []string{"app", "snapshot", "--owner", "nicerobot"},
			wantCfg:      domain.Config{Owner: "nicerobot", SettingsPath: ".github"},
			wantContains: `"owner": "nicerobot"`,
		},
		{
			name:    "missing required owner errors",
			args:    []string{"app", "snapshot"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			origRun, origCfg := runAction, cfg
			t.Cleanup(func() { runAction, cfg = origRun, origCfg })

			var gotCfg domain.Config
			runAction = func(_ context.Context, _ *slog.Logger, c domain.Config, _ ...string) (domain.Result, error) {
				gotCfg = c
				return domain.Result{Owner: string(c.Owner)}, nil
			}

			var stdout bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn}))
			appCmd := &cli.Command{
				Name:     "app",
				Writer:   &stdout,
				Commands: []*cli.Command{Command()},
				Metadata: map[string]any{app.LoggerMetadataKey: logger},
			}

			err := appCmd.Run(context.Background(), tt.args)
			if tt.wantErr {
				must.Error(err)
				return
			}
			must.NoError(err)
			want.Equal(tt.wantCfg, gotCfg)
			want.Contains(stdout.String(), tt.wantContains)
		})
	}
}
