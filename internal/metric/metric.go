package metric

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"

	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
)

var rawMetricOne = "raw_metric_one"

type accumulator struct {
	Selected int64
	Total    int64
}

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

	var lockGroup sync.WaitGroup
	lockGroup.Add(1)
	go calculateUptimeMetricOne(storage, rawMetricModel.UpTime, metricModel, &lockGroup)
	// TODO: Run a go routine to calculate the forwards_rating of the node
	// TODO: Run a go routine to calculate the rating about each channels.
	lockGroup.Wait()
	return nil
}

func calculateUptimeMetricOne(storage db.MetricsDatabase, nodeUpTime *RawPercentageData, metricModel *model.MetricOne, lock *sync.WaitGroup) {
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
		// This is a correct assumtion? or I need to iterate inside the DB?
		nodeUpTime.TodaySuccess = onlineUpdate
		nodeUpTime.TodayTotal = totUpdate
	}
	nodeUpTime.TodayTimestamp = lastTimestamp

	tenDaysStored := nodeUpTime.TenDaysTimestamp
	if utime.InRangeFromUnix(tenDaysStored, lastTimestamp, 10*24*time.Hour) {
		// accumulate
		nodeUpTime.TenDaysSuccess += onlineUpdate
		nodeUpTime.TenDaysTotal += totUpdate
	} else {
		// go back of 10 days
		// FIXME: Add this logic inside the utils module.
		firstDate := time.Unix(lastTimestamp, 0).Add(-10 * 24 * time.Hour).Unix()
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.TenDaysSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.TenDaysTotal = uint64(acc.Total) + totUpdate
		nodeUpTime.TenDaysTimestamp = firstDate
	}

	// 30 days
	thirtyDaysStored := nodeUpTime.ThirtyDaysTimestamp
	if utime.InRangeFromUnix(thirtyDaysStored, lastTimestamp, 30*24*time.Hour) {
		nodeUpTime.ThirtyDaysSuccess += onlineUpdate
		nodeUpTime.ThirtyDaysTotal += totUpdate
	} else {
		// go back of 10 days
		// FIXME: Add this logic inside the utils module.
		firstDate := time.Unix(lastTimestamp, 0).Add(-30 * 24 * time.Hour).Unix()
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.ThirtyDaysSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.ThirtyDaysTotal = uint64(acc.Total) + totUpdate
		nodeUpTime.ThirtyDaysTimestamp = firstDate
	}

	// 6 month
	sixMonthsStored := nodeUpTime.SixMonthsTimestamp
	if utime.InRangeFromUnix(sixMonthsStored, lastTimestamp, 6*30*24*time.Hour) {
		nodeUpTime.ThirtyDaysSuccess += onlineUpdate
		nodeUpTime.ThirtyDaysTotal += totUpdate
	} else {
		// go back of 10 days
		// FIXME: Add this logic inside the utils module.
		firstDate := time.Unix(lastTimestamp, 0).Add(6 * -30 * 24 * time.Hour).Unix()
		acc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		if err := walkThroughUpTime(storage, metricModel, firstDate, lastTimestamp, acc); err != nil {
			log.GetInstance().Errorf("Error during uptime iteration: %s", err)
		}
		nodeUpTime.SixMonthsSuccess = uint64(acc.Selected) + onlineUpdate
		nodeUpTime.SixMonthsTotal = uint64(acc.Total) + totUpdate
		nodeUpTime.SixMonthsTimestamp = firstDate
	}

	// full
	nodeUpTime.FullSuccess += onlineUpdate
	nodeUpTime.FullTotal += totUpdate
	lock.Done()
}

func accumulateUpTime(payloadStr string, acc *accumulator) error {
	var nodeMetric model.NodeMetric
	if err := json.Unmarshal([]byte(payloadStr), &nodeMetric); err != nil {
		return err
	}

	for _, upTimeItem := range nodeMetric.UpTime {
		if upTimeItem.Timestamp > 0 {
			acc.Selected++
		}
		acc.Total++
	}
	return nil
}

func walkThroughUpTime(storage db.MetricsDatabase, modelMetric *model.MetricOne, startDate int64, endDate int64, acc *accumulator) error {
	baseID, _ := storage.ItemID(modelMetric)
	startID := strings.Join([]string{baseID, fmt.Sprint(startDate), "metric"}, "/")
	endID := strings.Join([]string{baseID, fmt.Sprint(endDate), "metric"}, "/")
	err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
		if err := accumulateUpTime(itemValue, acc); err != nil {
			log.GetInstance().Errorf("Error during counting: %s", err)
			return err
		}
		return nil
	})
	return err
}
