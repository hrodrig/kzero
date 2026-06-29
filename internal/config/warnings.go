package config

// DeferredFeatureWarnings reports config fields accepted by the schema but not
// yet implemented by the v1 engine. Callers should print each message (e.g. to
// stderr) after a successful Load; they are warnings, not validation errors.
//
// Active deferred items (0.8.x band):
//   - run.api_watchdog.enabled (PR2 #39 schema; PR3 #36 engine)
//   - notify.require_delivery (PR2 #39 schema; requires #35 + PR3 #36 wiring)
func DeferredFeatureWarnings(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	if cfg.Run.APIWatchdog != nil && cfg.Run.APIWatchdog.Enabled {
		out = append(out,
			"run.api_watchdog.enabled: accepted by schema; engine watchdog goroutine is not implemented until v0.8.0 PR3 (#36).")
	}
	if cfg.Notify.RequireDelivery != nil && *cfg.Notify.RequireDelivery {
		out = append(out,
			"notify.require_delivery: accepted by schema; the engine does not yet fail pipelines when pipeline.error notify POSTs cannot be sent. Tracked for v0.8.0 alongside #35 and #36.")
	}
	return out
}
