package metric

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"

	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
)

var rawMetricOne = "raw_metric_one"

// Method to calculate the metric one output and store the result
// on the server
func CalculateMetricOneOutput(storage db.MetricsDatabase, metricModel *model.MetricOne) error {
	metricKey := strings.Join([]string{metricModel.NodeID, rawMetricOne}, "/")
	log.GetInstance().Infof("Raw metric key output is: %s", metricKey)
	rawMetric, err := storage.GetRawValue(metricKey)
	var rawMetricModel RawMetricOneOutput
	if err != nil {
		// If the metrics it is not found, create a new one
		// and continue to fill it.
		rawMetricModel = *NewRawMetricOneOutput(0)
	} else {
		if err := json.Unmarshal(rawMetric, &rawMetricModel); err != nil {
			return err
		}
	}

	go calculateUptimeMetricOne(rawMetricModel.UpTime, metricModel)
	// TODO: Run a go routine to calculate the forwards_rating of the node
	// TODO: Run a go routine to calculate the rating about each channels.
	return nil
}

func calculateUptimeMetricOne(nodeUpTime *RawPercentageData, metricModel *model.MetricOne) {
	// TODO: Get the timestamp from the metricModel
	// TODO: Update all the timestamp that we have in the nodeUpTime
	// log if some error happens
	totUpdate := uint64(0)
	onlineUpdate := uint64(0)
	lastTimestamp := int64(0)
	for _, upTime := range metricModel.UpTime {
		if upTime.Timestamp != 0 {
			onlineUpdate++
		}
		if int64(upTime.Timestamp) > lastTimestamp {
			lastTimestamp = int64(upTime.Timestamp)
		}
		totUpdate++
	}

	todayStored := nodeUpTime.TodayTimestamp
	if utime.SameDayUnix(todayStored, lastTimestamp) {
		//Accumulate
		nodeUpTime.TodaySuccess += onlineUpdate
		nodeUpTime.TodayTotal += totUpdate
	} else {
		// reset
		nodeUpTime.TodaySuccess = onlineUpdate
		nodeUpTime.TodayTotal = totUpdate
	}
	nodeUpTime.TodayTimestamp = lastTimestamp

	tenDaysStored := nodeUpTime.TenDaysTimestamp
	if utime.InRangeMonthsUnix(lastTimestamp, tenDaysStored, 1) {
		// accumulate
		nodeUpTime.TenDaysSuccess += onlineUpdate
		nodeUpTime.TenDaysTotal += totUpdate
	} else {
		firstDate := time.Unix(lastTimestamp, 0).Add(-1 * time.Hour).Unix()
		nodeUpTime.TenDaysTimestamp = firstDate
		// TODO: iterate inside the db to get all the information about
		// the sub
	}
}
