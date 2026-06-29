// Package watchdog implements the API reachability check described in
// docs/plan-0.8.x.md PR3 (#36). The engine starts a Watchdog goroutine
// during live runs when cfg.Run.APIWatchdog is configured; the goroutine
// probes the cluster API periodically and trips after the configured
// fail-after deadline of cumulative unreachability, at which point the
// watchdog calls an OnTrip callback so the engine can fail the step in
// flight. Watchdog does not own the cancel context; it observes one
// passed by the caller, which keeps responsibilities narrow.
//
// Design notes:
//
//  1. The probe function (Healthz) is injected by the caller; this keeps
//     the package free of client-go and trivially testable with a fake.
//
//  2. Time is injectable through a Clock interface so unit tests can
//     drive deterministic behavior.
//
//  3. Trip is one-shot and fires through OnTrip, which the engine uses
//     to cancel the active step's context.
//
//  4. Recoveries reset the failure clock — a transient blip shorter than
//     FailAfter does not trip the watchdog.
//
//  5. Config.Events (optional) exposes ProbeCalled and TripFired signals
//     so tests can synchronize without depending on goroutine scheduling.
//     Production paths leave it nil; the goroutine takes the events
//     channel's empty-write fast path.
package watchdog

import (
	"context"
	"sync"
	"time"
)

// Clock is the minimum surface of the standard time package that Watchdog
// uses. Tests can supply a fake to drive deterministic behavior; production
// callers pass realClock.
type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) Ticker
}

// Ticker is the minimum surface used by Watchdog. Mirrors time.Ticker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type realClock struct{}

func (realClock) Now() time.Time                   { return time.Now() }
func (realClock) NewTicker(d time.Duration) Ticker { return realTicker{t: time.NewTicker(d)} }

type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()               { r.t.Stop() }

// RealClock returns a Clock backed by the standard time package.
func RealClock() Clock { return realClock{} }

// EventType discriminates Event payloads.
type EventType int

const (
	// EventProbeCalled fires after every probe, whether successful or not.
	EventProbeCalled EventType = iota
	// EventTripFired fires exactly once when cumulative unreachability
	// crosses FailAfter. OnTrip is invoked immediately after.
	EventTripFired
)

// Event is a signal emitted by the Watchdog goroutine.
type Event struct {
	Type     EventType
	ProbeErr error // populated for EventProbeCalled
}

// Config configures a Watchdog. Zero-value is not usable.
type Config struct {
	// Interval is the period between Healthz probes. Must be > 0.
	Interval time.Duration
	// FailAfter is the cumulative unreachability deadline that trips
	// the watchdog. Must be > 0. Typical values mirror
	// docs/plan-0.8.x.md (#36 default 5m).
	FailAfter time.Duration
	// Healthz is invoked on each probe. It MUST honor the passed ctx and
	// return promptly when ctx is canceled. Errors count as a failed
	// probe; nil counts as a successful probe.
	Healthz func(ctx context.Context) error
	// OnTrip is invoked at most once, after cumulative unreachability
	// exceeds FailAfter. It runs in the watchdog goroutine; callers
	// should keep it short — typically, cancel the parent step context.
	OnTrip func()
	// Clock is used for time and the ticker. When nil, RealClock is used.
	Clock Clock
	// Events, when non-nil, receives EventProbeCalled and EventTripFired
	// notifications from the watchdog goroutine. Useful for tests; nil
	// keeps the production path allocation-free.
	Events chan<- Event
}

// Watchdog runs an API reachability check on a goroutine until ctx is
// canceled or the trip fires once. The zero value is not usable; build
// one with New.
type Watchdog struct {
	cfg      Config
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once
}

// New returns a started Watchdog. Call Stop (or cancel the ctx you pass
// to Run) to terminate. New panics if cfg is invalid (zero Interval, zero
// FailAfter, nil Healthz).
func New(ctx context.Context, cfg Config) *Watchdog {
	if cfg.Interval <= 0 {
		panic("watchdog: Interval must be > 0")
	}
	if cfg.FailAfter <= 0 {
		panic("watchdog: FailAfter must be > 0")
	}
	if cfg.Healthz == nil {
		panic("watchdog: Healthz is required")
	}
	if cfg.Clock == nil {
		cfg.Clock = RealClock()
	}

	ctx, cancel := context.WithCancel(ctx)
	w := &Watchdog{
		cfg:    cfg,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go w.run(ctx)
	return w
}

// Stop terminates the watchdog and waits for the goroutine to exit. Safe
// to call multiple times; subsequent calls are no-ops.
func (w *Watchdog) Stop() {
	w.stopOnce.Do(func() { w.cancel() })
	<-w.done
}

func (w *Watchdog) run(parent context.Context) {
	defer close(w.done)

	ticker := w.cfg.Clock.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	var (
		firstFailure time.Time
		failed       bool
		tripped      bool
		probeBudget  = w.cfg.Interval
	)

	for {
		select {
		case <-parent.Done():
			return
		case <-ticker.C():
			probeCtx, probeCancel := context.WithTimeout(parent, probeBudget)
			err := w.cfg.Healthz(probeCtx)
			probeCancel()
			w.emit(Event{Type: EventProbeCalled, ProbeErr: err})
			now := w.cfg.Clock.Now()
			if err == nil {
				failed = false
				firstFailure = time.Time{}
				continue
			}
			if !failed {
				failed = true
				firstFailure = now
				continue
			}
			if !tripped && now.Sub(firstFailure) >= w.cfg.FailAfter {
				tripped = true
				w.emit(Event{Type: EventTripFired})
				if w.cfg.OnTrip != nil {
					w.cfg.OnTrip()
				}
			}
		}
	}
}

func (w *Watchdog) emit(e Event) {
	if w.cfg.Events == nil {
		return
	}
	select {
	case w.cfg.Events <- e:
	default:
	}
}
