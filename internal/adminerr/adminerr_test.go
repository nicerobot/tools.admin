package adminerr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/adminerr"
)

func TestErrorString(t *testing.T) {
	assert.Equal(t, "GH_TOKEN environment variable must be set", adminerr.ErrNoToken.Error())
}

func TestWithNilCauseNoArgs(t *testing.T) {
	err := adminerr.ErrNoToken.With(nil)
	require.ErrorIs(t, err, adminerr.ErrNoToken)
	assert.Equal(t, "GH_TOKEN environment variable must be set", err.Error())
}

func TestWithCauseWraps(t *testing.T) {
	cause := errors.New("boom")
	err := adminerr.ErrHTTPStatus.With(cause)
	require.ErrorIs(t, err, adminerr.ErrHTTPStatus)
	require.ErrorIs(t, err, cause)
}

func TestWithArgsAppendsSpaceSeparated(t *testing.T) {
	err := adminerr.ErrHTTPStatus.With(nil, "status", 404)
	require.ErrorIs(t, err, adminerr.ErrHTTPStatus)
	assert.Equal(t, "unexpected GitHub API status: status 404", err.Error())
}

func TestWithCauseAndArgs(t *testing.T) {
	cause := errors.New("boom")
	err := adminerr.ErrCommand.With(cause, "args", "git status")
	require.ErrorIs(t, err, adminerr.ErrCommand)
	require.ErrorIs(t, err, cause)
	assert.Equal(t, "command failed: boom: args git status", err.Error())
}
