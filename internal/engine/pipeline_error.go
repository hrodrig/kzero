package engine

import "fmt"

// PipelineError describes a pipeline step or phase hook failure for operator surfaces.
type PipelineError struct {
	Phase string
	Hook  string
	Index int
	Ref   string
	Err   error
}

func (e *PipelineError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Hook != "" {
		return fmt.Sprintf("hook %s: %v", e.Hook, e.Err)
	}
	if e.Ref != "" {
		return fmt.Sprintf("step %s (%s[%d]): %v", e.Ref, e.Phase, e.Index, e.Err)
	}
	if e.Phase != "" && e.Index >= 0 {
		return fmt.Sprintf("step %s[%d]: %v", e.Phase, e.Index, e.Err)
	}
	return e.Err.Error()
}

func (e *PipelineError) Unwrap() error {
	return e.Err
}

// FailedStep returns a compact label for notify payloads.
func (e *PipelineError) FailedStep() string {
	if e == nil {
		return ""
	}
	if e.Hook != "" {
		return "hook:" + e.Hook
	}
	if e.Ref != "" {
		return e.Ref
	}
	if e.Phase != "" && e.Index >= 0 {
		return fmt.Sprintf("%s[%d]", e.Phase, e.Index)
	}
	return ""
}
