package engine

// Phase identifies a pipeline direction for logging and runner dispatch.
type Phase string

const (
	PhaseDown Phase = "down"
	PhaseUp   Phase = "up"
)
