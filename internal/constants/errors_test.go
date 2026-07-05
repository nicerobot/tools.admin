package constants

import (
	"errors"
	"testing"

	errs "github.com/gomatic/go-error"
	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	t.Parallel()
	assert.New(t).Equal("command failed", ErrCommand.Error())
}

func TestError_With(t *testing.T) {
	t.Parallel()
	cause := errors.New("disk full")

	tests := []struct {
		err         error
		name        string
		sentinel    errs.Const
		wantMessage string
		args        []any
		wantIs      []error
	}{
		{
			name:        "sentinel only",
			sentinel:    ErrNoAuth,
			err:         nil,
			args:        nil,
			wantIs:      []error{ErrNoAuth},
			wantMessage: "GH_TOKEN environment variable must be set",
		},
		{
			name:        "wraps cause",
			sentinel:    ErrWriteFile,
			err:         cause,
			args:        nil,
			wantIs:      []error{ErrWriteFile, cause},
			wantMessage: "failed to write override file: disk full",
		},
		{
			name:        "appends args",
			sentinel:    ErrStaleRepoExists,
			err:         nil,
			args:        []any{"repo", "owner/x"},
			wantIs:      []error{ErrStaleRepoExists},
			wantMessage: "repo exists but was not returned by list_repos; aborting to prevent data loss: repo owner/x",
		},
		{
			name:        "wraps cause and appends args",
			sentinel:    ErrRemoveFile,
			err:         cause,
			args:        []any{"file", "/tmp/x"},
			wantIs:      []error{ErrRemoveFile, cause},
			wantMessage: "failed to remove override file: disk full: file /tmp/x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := assert.New(t)

			err := tt.sentinel.With(tt.err, tt.args...)
			for _, target := range tt.wantIs {
				want.ErrorIs(err, target)
			}
			want.Equal(tt.wantMessage, err.Error())
		})
	}
}
