package service

import (
	"db-mcp/internal/config"
	"db-mcp/internal/errors"
)

// ValidateDeleteMode keeps physical deletion opt-in at both the server and request levels.
func ValidateDeleteMode(cfg *config.Config, physical bool) error {
	if physical && (cfg == nil || !cfg.AllowPhysicalDelete) {
		return errors.NewError(errors.ErrInvalidInput, "physical deletion is disabled by configuration", nil)
	}
	return nil
}
