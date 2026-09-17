package createpr

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicerobot/tools.admin/internal/constants"
)

// TestMain enables the command for the package's tests before any of them start, so the
// behaviour behind the guard stays verified while the shipped binary refuses to run.
func TestMain(m *testing.M) {
	isEnabled = true
	os.Exit(m.Run())
}

// TestRunRefusesWhileDisabled is deliberately not parallel: it flips the package gate,
// and sequential tests finish before any parallel test resumes.
func TestRunRefusesWhileDisabled(t *testing.T) {
	must := require.New(t)
	orig := deps
	isBuilt := false
	deps = func() dependencies {
		isBuilt = true
		return dependencies{}
	}
	isEnabled = false
	t.Cleanup(func() {
		isEnabled = true
		deps = orig
	})

	_, err := Run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), Config{})

	must.ErrorIs(err, constants.ErrCreatePRDisabled)
	must.False(isBuilt, "a disabled run must not build any dependency")
}
