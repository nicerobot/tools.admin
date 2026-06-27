// Package createpr orchestrates the create-pr command: configure the bot
// identity, checkout the snapshot branch, stage the repos directory, and — only
// when something is staged — commit, force-push, and open a PR if none is open.
//
// It defines the command's Config (the flags the CLI binds) and Run (the
// orchestration entry point). Run delegates every git/gh operation to the
// reusable internal/gitcmd package and holds no CLI or output-formatting logic.
// This is the domain tier between the app tier (internal/app/commands/createpr)
// and the implementation tier (internal/gitcmd).
package createpr
