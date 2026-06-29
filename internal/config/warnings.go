package config

// DeferredFeatureWarnings reports config fields accepted by the schema but not
// yet implemented by the v1 engine. Callers should print each message (e.g. to
// stderr) after a successful Load; they are warnings, not validation errors.
//
// Active deferred items:
//   - notify.require_delivery (schema since v0.8.0; engine fail-fast on dispatch error not wired yet)
func DeferredFeatureWarnings(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	if cfg.Notify.RequireDelivery != nil && *cfg.Notify.RequireDelivery {
		out = append(out,
			"notify.require_delivery: accepted by schema; the engine logs [ERR] on failed notify POSTs (#35) but does not yet fail the pipeline when pipeline.error delivery cannot be sent.")
	}
	return out
}
