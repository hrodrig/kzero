package config

// DeferredFeatureWarnings reports config fields accepted by the schema but not
// yet implemented by the v1 engine. Callers should print each message (e.g. to
// stderr) after a successful Load; they are warnings, not validation errors.
func DeferredFeatureWarnings(cfg *Config) []string {
	return nil
}
