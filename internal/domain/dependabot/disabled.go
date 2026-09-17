package dependabot

import "github.com/nicerobot/tools.admin/internal/constants"

// isEnabled gates the command. The sweep maintains the fleet: it merges Dependabot pull requests across every owner. Fleet maintenance is switched off, so the zero
// value (false) keeps Run from doing anything at all; re-enabling is setting this to true.
var isEnabled bool

// guard refuses every run while the command is disabled, before any dependency is built.
func guard() error {
	if !isEnabled {
		return constants.ErrDependabotDisabled
	}
	return nil
}
