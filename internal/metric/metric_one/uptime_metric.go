package metric_one

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
	"time"
)

// CalculateUptimeMetricOneSync Execute the uptime rating of the node
// TODO: Refactoring this logic and use the channels
func CalculateUptimeMetricOneSync(storage db.MetricsDatabase, rawMetric *RawMetricOneOutput,
	metricModel *model.MetricOne) {

	onlineUpdate := uint64(0)
	lastTimestamp := int64(-1)

	for _, upTime := range metricModel.UpTime {
		// We take in consideration only the on_update that are generated by the
		// plugin timeout.
		// we could make this calculation better to take in account also the
		// previous event, but for now we implement the easy things
		// FIXME: in this case we can do also a more complex estimation if we take
		// in consideration also the prev status
		if upTime.Timestamp != 0 && upTime.Event == "on_update" {
			onlineUpdate++
		}
		if int64(upTime.Timestamp) > lastTimestamp {
			lastTimestamp = int64(upTime.Timestamp)
		}

	}

	nodeUpTime := rawMetric.UpTime
	todayStored := nodeUpTime.TodayTimestamp
	if utime.SameDayUnix(todayStored, lastTimestamp) {
		//Accumulate
		nodeUpTime.TodaySuccess += onlineUpdate
	} else {
		// FIXME: This is a correct assumption? or I need to iterate inside the DB?
		nodeUpTime.TodaySuccess = onlineUpdate
	}
	nodeUpTime.TodayTotal = TodayOccurrence
	nodeUpTime.TodayTimestamp = lastTimestamp

	tenDaysStored := nodeUpTime.TenDaysTimestamp
	if utime.InRangeFromUnix(lastTimestamp, tenDaysStored, 10*24*time.Hour) {
		// accumulate
		nodeUpTime.TenDaysSuccess += onlineUpdate
	} else {
		firstDate := utime.SubToTimestamp(lastTimestamp, 10*24*time.Hour)
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.TenDaysSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.TenDaysTimestamp = firstDate
	}
	nodeUpTime.TenDaysTotal = RenDaysOccurrence

	// 30 days
	thirtyDaysStored := nodeUpTime.ThirtyDaysTimestamp
	if utime.InRangeFromUnix(lastTimestamp, thirtyDaysStored, 30*24*time.Hour) {
		nodeUpTime.ThirtyDaysSuccess += onlineUpdate
	} else {
		firstDate := utime.SubToTimestamp(lastTimestamp, 30*24*time.Hour)
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.ThirtyDaysSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.ThirtyDaysTimestamp = firstDate
	}
	nodeUpTime.ThirtyDaysTotal = ThirtyDaysOccurrence

	// 6 month
	sixMonthsStored := nodeUpTime.SixMonthsTimestamp
	if utime.InRangeFromUnix(lastTimestamp, sixMonthsStored, 6*30*24*time.Hour) {
		nodeUpTime.SixMonthsSuccess += onlineUpdate
	} else {
		firstDate := utime.SubToTimestamp(lastTimestamp, 6*30*24*time.Hour)
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.SixMonthsSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.SixMonthsTimestamp = firstDate
	}
	nodeUpTime.SixMonthsTotal = SixMonthsOccurrence

	// full
	nodeUpTime.FullSuccess += onlineUpdate
	nodeUpTime.FullTotal = utime.OccurenceInUnixRange(rawMetric.Age, lastTimestamp, 30*time.Minute)
}
