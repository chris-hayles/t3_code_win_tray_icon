//go:build windows

package winutil

import "time"

func SleepMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
