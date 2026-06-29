package app

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"
)

// Runner is the signature every domain Run satisfies: it takes a context, a
// logger, a typed config, and the positional arguments, and returns a typed
// result. The app tier knows nothing about a command beyond this shape.
type Runner[CONFIG any, RESULT any] func(context.Context, *slog.Logger, CONFIG, ...string) (RESULT, error)

// getLogger is indirected through a variable so tests can substitute it.
var getLogger = GetLogger

// action runs a domain runner and renders its result. It is the usual seam
// between the CLI framework and the domain tier; rendering the result is the
// default, not a requirement — a command that produces its own output (an
// interactive REPL, say) may bind its Run without rendering.
func action[CONFIG, RESULT any](
	ctx context.Context,
	command *cli.Command,
	cfg CONFIG,
	run Runner[CONFIG, RESULT],
) error {
	logger := getLogger(command)

	result, err := run(ctx, logger, cfg, command.Args().Slice()...)
	if err != nil {
		return err
	}

	return output(command.Root().Writer, formatJSON, result)
}

// Default binds a config pointer and a domain runner into a cli.Command action.
// Most commands declare `Action: app.Default(&cfg, domain.Run)` and nothing
// more; a command that renders its own output may bind its Run through a
// non-rendering action instead.
func Default[CONFIG, RESULT any](cfg *CONFIG, run Runner[CONFIG, RESULT]) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		return action(ctx, command, *cfg, run)
	}
}
