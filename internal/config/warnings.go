package config

// DeferredFeatureWarnings reports config fields accepted by the schema but not
// yet implemented by the v1 engine. Callers should print each message (e.g. to
// stderr) after a successful Load; they are warnings, not validation errors.
func DeferredFeatureWarnings(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	if cfg.Notify.Slack.Enabled {
		out = append(out, "notify.slack.enabled is true but Slack notifications are not implemented yet")
	}
	if cfg.Notify.Discord.Enabled {
		out = append(out, "notify.discord.enabled is true but Discord notifications are not implemented yet")
	}
	return out
}
