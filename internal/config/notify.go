package config

// RequireNotifyDelivery reports whether a failed pipeline.error or
// pipeline.stalled notify POST must fail the pipeline.
func RequireNotifyDelivery(cfg *Config) bool {
	return cfg != nil && cfg.Notify.RequireDelivery != nil && *cfg.Notify.RequireDelivery
}
