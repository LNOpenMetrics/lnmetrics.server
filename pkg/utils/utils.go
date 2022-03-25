// Package utils / Simple utils function used int he application
/// with scope only inside the application
// BTW, we make this public if can be useful for someone else.
package utils

import (
	"time"

	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
)

// ReturnOccurrence return how mayne occurrence are container in a period of time
// with a specified duration step.
func ReturnOccurrence(period time.Duration, step time.Duration) uint64 {
	now := time.Now()
	after := now.Add(period)
	return utime.OccurenceInUnixRange(now.Unix(), after.Unix(), step)
}
