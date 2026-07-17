package api

import log "github.com/sirupsen/logrus"

// logExternalPricingError emits a warning through the shared logrus logger so
// operators can spot malformed pricing files at startup without crashing the
// service. Kept in its own file so tests can substitute a no-op logger.
func logExternalPricingError(path string, err error) {
	log.Warnf("external pricing file at %s failed to parse: %v (falling back to built-in table)", path, err)
}
