package backoff

import (
	"testing"
	"time"
)

func TestDelay(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 15 * time.Second},
		{99, 15 * time.Second},
	}
	for _, c := range cases {
		if got := Delay(c.attempt); got != c.want {
			t.Fatalf("Delay(%d)=%v want %v", c.attempt, got, c.want)
		}
	}
}

func TestShouldResetUptime(t *testing.T) {
	if !ShouldResetUptime(60 * time.Second) {
		t.Fatal("60s should reset")
	}
	if ShouldResetUptime(59*time.Second + 999*time.Millisecond) {
		t.Fatal("just under 60s should not reset")
	}
	if !ShouldResetUptime(5 * time.Minute) {
		t.Fatal("longer uptime should reset")
	}
}
