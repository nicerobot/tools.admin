package snapshot

import "github.com/nicerobot/tools.admin/internal/repo"

// Config holds the flags for the snapshot command. Its fields are bound by the
// CLI tier and read by Run; it carries no behavior. Both fields reuse the
// implementation tier's named types, so no domain-local types.go is needed.
type Config struct {
	Owner        repo.Owner
	SettingsPath repo.SettingsPath
}
