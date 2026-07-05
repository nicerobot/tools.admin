package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/nicerobot/tools.admin/internal/constants"
)

type (
	testConfig struct{}
	testResult struct {
		Value string `json:"value"`
	}
)

func TestOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wantErr      error
		name         string
		format       format
		wantContains string
	}{
		{name: "json", format: formatJSON, wantContains: `"value": "x"`},
		{name: "yaml", format: formatYAML, wantContains: "value: x"},
		{name: "empty defaults to json", format: format(""), wantContains: `"value": "x"`},
		{name: "invalid format", format: format("toml"), wantErr: constants.ErrInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			var buf bytes.Buffer
			err := output(&buf, tt.format, testResult{Value: "x"})

			if tt.wantErr != nil {
				must.Error(err)
				want.ErrorIs(err, tt.wantErr)
				return
			}

			must.NoError(err)
			want.Contains(buf.String(), tt.wantContains)
		})
	}
}

func TestDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wantErr      error
		run          Runner[testConfig, testResult]
		name         string
		wantContains string
	}{
		{
			name: "success renders json result",
			run: func(context.Context, *slog.Logger, testConfig, ...string) (testResult, error) {
				return testResult{Value: "ok"}, nil
			},
			wantContains: `"value": "ok"`,
		},
		{
			name: "runner error propagates",
			run: func(context.Context, *slog.Logger, testConfig, ...string) (testResult, error) {
				return testResult{}, constants.ErrNoTarget
			},
			wantErr: constants.ErrNoTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, must := assert.New(t), require.New(t)

			var out bytes.Buffer
			cfg := testConfig{}
			command := &cli.Command{
				Name:   "x",
				Writer: &out,
				Action: Default(&cfg, tt.run),
			}

			err := command.Run(context.Background(), []string{"x"})

			if tt.wantErr != nil {
				must.Error(err)
				want.ErrorIs(err, tt.wantErr)
				return
			}

			must.NoError(err)
			want.Contains(out.String(), tt.wantContains)
		})
	}
}

func TestParseLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		level LogLevel
		want  slog.Level
	}{
		{name: "debug", level: "debug", want: slog.LevelDebug},
		{name: "error", level: "error", want: slog.LevelError},
		{name: "empty defaults to info", level: "", want: slog.LevelInfo},
		{name: "unknown defaults to info", level: "bogus", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.New(t).Equal(tt.want, parseLevel(tt.level))
		})
	}
}

func TestNewLogger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		cfg          LoggerConfig
		wantContains string
	}{
		{name: "text format", cfg: LoggerConfig{LogFormat: "text", LogLevel: "info"}, wantContains: "msg=hello"},
		{name: "json format", cfg: LoggerConfig{LogFormat: "json", LogLevel: "info"}, wantContains: `"msg":"hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := assert.New(t)

			var buf bytes.Buffer
			logger := NewLogger(&buf, tt.cfg)

			logger.Info("hello")
			want.Contains(buf.String(), tt.wantContains)
		})
	}
}

func TestHandlerTypeFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		format LogFormat
		want   handlerType
	}{
		{name: "json", format: "json", want: handlerTypeJSON},
		{name: "text", format: "text", want: handlerTypeText},
		{name: "empty defaults to text", format: "", want: handlerTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.New(t).Equal(tt.want, handlerTypeFor(tt.format))
		})
	}
}

func TestGetLogger(t *testing.T) {
	t.Parallel()

	t.Run("nil command returns default", func(t *testing.T) {
		assert.New(t).NotNil(GetLogger(nil))
	})

	t.Run("command without logger returns default", func(t *testing.T) {
		command := &cli.Command{Name: "x"}
		assert.New(t).NotNil(GetLogger(command))
	})

	t.Run("command with logger returns it", func(t *testing.T) {
		want := assert.New(t)

		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
		command := &cli.Command{
			Name:     "x",
			Metadata: map[string]any{LoggerMetadataKey: logger},
		}

		want.Same(logger, GetLogger(command))
	})
}
