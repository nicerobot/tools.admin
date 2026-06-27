// Package snapshot orchestrates the snapshot command: load the org/account
// defaults, list every live repo, diff each into a repos/<name>.yml override
// file, and reconcile stale files — but only after verifying every stale
// candidate is truly gone, so a token that cannot see all repos can never delete
// a file it merely failed to list.
//
// It defines the command's Config (the flags the CLI binds) and Run (the
// orchestration entry point). Run validates input and composes the reusable
// internal/github, internal/settings, and internal/overrides packages; it holds
// no CLI, flag, or output-formatting logic. This is the domain tier between the
// app tier (internal/app/commands/snapshot) and the implementation tier.
package snapshot
