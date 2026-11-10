//go:build windows

package supervisor

import (
	"testing"
	"time"

	"t3tray/internal/backoff"
	"t3tray/internal/updatebusy"
)

// A server that starts successfully but exits right away must be restarted with
// backoff, not in a tight loop.
func TestLoop_BacksOffAfterShortLivedSuccessfulStart(t *testing.T) {
	s := &Supervisor{stopCh: make(chan struct{})}

	var (
		now    = time.Unix(1_000_000, 0)
		starts int
		sleeps []time.Duration
	)
	s.hooks = &loopHooks{
		cliPresent: func() bool { return true },
		start: func() bool {
			starts++
			s.mu.Lock()
			s.startedAt = now
			s.mu.Unlock()
			return true
		},
		wait: func() {
			// The wrapper exits 2s after it started.
			now = now.Add(2 * time.Second)
		},
		foreignAfter: func() bool { return false },
		now:          func() time.Time { return now },
		sleep: func(d time.Duration) bool {
			sleeps = append(sleeps, d)
			// Stop after four restarts so the loop returns.
			return len(sleeps) < 5
		},
	}

	s.loop()

	want := []time.Duration{backoff.Delay(1), backoff.Delay(2), backoff.Delay(3), backoff.Delay(4), backoff.Delay(5)}
	if starts != len(want) {
		t.Fatalf("starts = %d, want %d", starts, len(want))
	}
	if len(sleeps) != len(want) {
		t.Fatalf("sleeps = %v, want %v", sleeps, want)
	}
	for i := range want {
		if sleeps[i] != want[i] {
			t.Fatalf("sleeps = %v, want %v", sleeps, want)
		}
	}
}

// While a T3 Code update is busy, the loop must not start serve or advance backoff.
func TestLoop_PausesWhileUpdateBusy(t *testing.T) {
	s := &Supervisor{stopCh: make(chan struct{})}

	var (
		starts int
		polls  int
		sleeps []time.Duration
	)
	s.hooks = &loopHooks{
		cliPresent: func() bool { return true },
		updateBusy: func() (bool, string) {
			polls++
			if polls <= 3 {
				return true, "T3 Code installer is running"
			}
			return false, ""
		},
		start: func() bool {
			starts++
			s.mu.Lock()
			s.startedAt = time.Unix(1_700_000_000, 0)
			s.mu.Unlock()
			close(s.stopCh)
			return true
		},
		wait: func() {
			// Short-lived so ShouldResetUptime does not clear attempt.
		},
		foreignAfter: func() bool { return false },
		now:          func() time.Time { return time.Unix(1_700_000_002, 0) },
		sleep: func(d time.Duration) bool {
			sleeps = append(sleeps, d)
			return true
		},
	}

	s.loop()

	if starts != 1 {
		t.Fatalf("starts = %d, want 1 after update cleared", starts)
	}
	if polls < 3 {
		t.Fatalf("polls = %d, want at least 3 busy polls", polls)
	}
	for i := 0; i < 3 && i < len(sleeps); i++ {
		if sleeps[i] != updatebusy.PollWhileBusy {
			t.Fatalf("sleeps[%d] = %v, want %v", i, sleeps[i], updatebusy.PollWhileBusy)
		}
	}
	s.mu.Lock()
	attempt := s.attempt
	s.mu.Unlock()
	if attempt != 1 {
		t.Fatalf("attempt = %d, want 1 (busy polls must not advance backoff)", attempt)
	}
}

// A run that lasted long enough resets the counter and restarts without delay.
func TestLoop_NoBackoffAfterLongRun(t *testing.T) {
	s := &Supervisor{stopCh: make(chan struct{})}

	now := time.Unix(1_000_000, 0)
	starts := 0
	var sleeps []time.Duration
	s.hooks = &loopHooks{
		cliPresent: func() bool { return true },
		start: func() bool {
			starts++
			s.mu.Lock()
			s.startedAt = now
			s.mu.Unlock()
			return true
		},
		wait: func() {
			now = now.Add(2 * time.Minute)
			if starts == 2 {
				close(s.stopCh)
			}
		},
		foreignAfter: func() bool { return false },
		now:          func() time.Time { return now },
		sleep: func(d time.Duration) bool {
			sleeps = append(sleeps, d)
			return true
		},
	}

	s.loop()

	if len(sleeps) != 0 {
		t.Fatalf("unexpected sleeps after long run: %v", sleeps)
	}
	s.mu.Lock()
	attempt := s.attempt
	s.mu.Unlock()
	if attempt != 0 {
		t.Fatalf("attempt = %d, want 0 after reset", attempt)
	}
}
