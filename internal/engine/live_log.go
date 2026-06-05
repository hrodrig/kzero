package engine

import "fmt"

func (r *LiveRunner) logLive(format string, args ...interface{}) {
	if r.Log == nil {
		return
	}
	r.Log.Live(fmt.Sprintf(format, args...))
}
