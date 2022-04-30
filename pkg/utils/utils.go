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

// Percentage return the percentage of success.
// FIXME: this need to return a float
func Percentage(selected uint64, total uint64) int {
	if total == 0 {
		return 0
	}
	percentage := int((float64(selected) / float64(total)) * 100)
	// this can happen when the update are trigger not each 30 minutes
	// FIXME: this need to be fixed!
	if percentage > 100 {
		percentage = 100
	}
	return percentage
}
