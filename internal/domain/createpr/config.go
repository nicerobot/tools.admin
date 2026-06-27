package createpr

import "github.com/nicerobot/tools.admin/internal/repo"

// Config holds the flags for the create-pr command. Its fields are bound by the
// CLI tier and read by Run; it carries no behavior. Every field reuses the
// implementation tier's named types, so no domain-local types.go is needed.
type Config struct {
	SettingsPath repo.SettingsPath
	Branch       repo.Branch
	Base         repo.Base
}
