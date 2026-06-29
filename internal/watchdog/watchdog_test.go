package watchdog

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdog_noTripWhenHealthzAlwaysOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var trips atomic.Int32
	w := New(ctx, Config{
		Interval:  time.Millisecond,
		FailAfter: 20 * time.Millisecond,
		Healthz:   func(context.Context) error { return nil },
		OnTrip:    func() { trips.Add(1) },
	})
	defer w.Stop()

	time.Sleep(50 * time.Millisecond)

	if got := trips.Load(); got != 0 {
		t.Fatalf("trips = %d, want 0", got)
	}
}

func TestWatchdog_tripsAfterCumulativeFailAfter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var trips atomic.Int32
	done := make(chan struct{})

	w := New(ctx, Config{
		Interval:  time.Millisecond,
		FailAfter: 5 * time.Millisecond,
		Healthz:   func(context.Context) error { return errors.New("fail") },
		OnTrip: func() {
			trips.Add(1)
			close(done)
		},
	})
	defer w.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("OnTrip did not fire; trips=%d", trips.Load())
	}

	if got := trips.Load(); got != 1 {
		t.Fatalf("trips = %d, want 1", got)
	}
}

func TestWatchdog_onTripFiresExactlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var trips atomic.Int32

	w := New(ctx, Config{
		Interval:  time.Millisecond,
		FailAfter: 3 * time.Millisecond,
		Healthz:   func(context.Context) error { return errors.New("fail") },
		OnTrip:    func() { trips.Add(1) },
	})
	defer w.Stop()

	time.Sleep(50 * time.Millisecond)

	if got := trips.Load(); got != 1 {
		t.Fatalf("trips = %d, want 1", got)
	}
}

func TestWatchdog_recoversBeforeFailAfter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var fails int32 = 2
	var trips atomic.Int32

	w := New(ctx, Config{
		Interval:  time.Millisecond,
		FailAfter: 50 * time.Millisecond,
		Healthz: func(context.Context) error {
			if atomic.AddInt32(&fails, -1) >= 0 {
				return errors.New("transient")
			}
			return nil
		},
		OnTrip: func() { trips.Add(1) },
	})
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	if got := trips.Load(); got != 0 {
		t.Fatalf("trips = %d, want 0 (recovery reset)", got)
	}
}

func TestWatchdog_cancelCtxTerminatesWithoutTrip(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())

	var trips atomic.Int32

	w := New(parentCtx, Config{
		Interval:  time.Millisecond,
		FailAfter: 50 * time.Millisecond,
		Healthz:   func(context.Context) error { return errors.New("fail") },
		OnTrip:    func() { trips.Add(1) },
	})

	time.Sleep(5 * time.Millisecond)
	parentCancel()
	w.Stop()

	if got := trips.Load(); got != 0 {
		t.Fatalf("trips = %d, want 0", got)
	}
}

func TestWatchdog_panicsOnZeroConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
	}{
		{"zero interval", Config{FailAfter: time.Minute, Healthz: func(context.Context) error { return nil }}},
		{"zero fail_after", Config{Interval: time.Minute, Healthz: func(context.Context) error { return nil }}},
		{"nil Healthz", Config{Interval: time.Minute, FailAfter: time.Minute}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for %s", tc.name)
				}
			}()
			New(context.Background(), tc.cfg)
		})
	}
}
