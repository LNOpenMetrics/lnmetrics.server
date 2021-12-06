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
	lockGroup.Add(2)
	go calculateUptimeMetricOne(storage, rawMetricModel.UpTime, metricModel, &lockGroup)
	go calculateForwardsRatingMetricOne(storage, rawMetricModel.ForwardsRating, metricModel, &lockGroup)
	// TODO: Run a go routine to calculate the rating about each channels.
	lockGroup.Wait()
	return nil
}

// Execute the uptime rating of the node
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

// Get last timestamp inside the uptime
func getLatestTimestamp(nodeUpTime []*model.Status) int64 {
	lastTimestamp := int64(0)
	for _, upTime := range nodeUpTime {
		if int64(upTime.Timestamp) > lastTimestamp {
			lastTimestamp = int64(upTime.Timestamp)
		}
	}
	return lastTimestamp
}

// Util function to iterate through the database item and accumulate the value
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

// Utils function to execute the logic of rating a node uptime, this is used to refactoring the logic in one
// compact method.
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

type forwardsAccumulator struct {
	Success     uint64
	Failed      uint64
	LocalFailed uint64
}

func calculateForwardsRatingMetricOne(storage db.MetricsDatabase, forwardsRating *RawForwardsRating,
	metricModel *model.MetricOne, lock *sync.WaitGroup) {

	acc := countForwards(metricModel.ChannelsInfo)
	lastTimestamp := getLatestTimestamp(metricModel.UpTime)
	calculateForwardsRatingByTimestamp(storage, metricModel, forwardsRating, acc, lastTimestamp)
	lock.Done()
}

func countForwards(channelsInfo []*model.StatusChannel) *forwardsAccumulator {
	acc := &forwardsAccumulator{
		Success:     0,
		Failed:      0,
		LocalFailed: 0,
	}

	for _, channelInfo := range channelsInfo {
		for _, forward := range channelInfo.Forwards {
			switch forward.Status {
			case "settled", "offered":
				acc.Success++
			case "failed":
				acc.Failed++
			case "local_failed":
				acc.LocalFailed++
			default:
				log.GetInstance().Errorf("The status %s it is not recognized", forward.Status)
			}
		}
	}
	return acc
}

type wrapperRawForwardRating struct {
	Wrapper   *RawForwardRating
	Timestamp int64
}

func calculateForwardsRatingByTimestamp(storage db.MetricsDatabase, metricModel *model.MetricOne,
	forwardsRating *RawForwardsRating, acc *forwardsAccumulator, actualTimestamp int64) {
	todayChan := make(chan *wrapperRawForwardRating)
	tenDaysChan := make(chan *wrapperRawForwardRating)
	thirtyDaysChan := make(chan *wrapperRawForwardRating)
	sixMonthsChan := make(chan *wrapperRawForwardRating)

	// call for today rating
	go calculateForwardRatingByPeriod(storage, metricModel, forwardsRating.TodayRating, acc,
		actualTimestamp, forwardsRating.TodayTimestamp, 1*24*time.Hour, todayChan)

	// call for last 10 days rating
	go calculateForwardRatingByPeriod(storage, metricModel, forwardsRating.TenDaysRating, acc,
		actualTimestamp, forwardsRating.TenDaysTimestamp, 10*24*time.Hour, tenDaysChan)

	// call for the last 30 days
	go calculateForwardRatingByPeriod(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.ThirtyDaysTimestamp, 30*24*time.Hour, thirtyDaysChan)

	// call for the last 6 months
	go calculateForwardRatingByPeriod(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.SixMonthsTimestamp, 6*utime.Month, sixMonthsChan)

	forwardsRating.FullRating.Success += acc.Success
	forwardsRating.FullRating.Failure += acc.Failed
	forwardsRating.FullRating.InternalFailure += acc.LocalFailed

	select {
	case today := <-todayChan:
		forwardsRating.TodayRating = today.Wrapper
		forwardsRating.TodayTimestamp = today.Timestamp
	case tenDays := <-tenDaysChan:
		forwardsRating.TenDaysRating = tenDays.Wrapper
		forwardsRating.TenDaysTimestamp = tenDays.Timestamp
	case thirtyDays := <-thirtyDaysChan:
		forwardsRating.ThirtyDaysRating = thirtyDays.Wrapper
		forwardsRating.ThirtyDaysTimestamp = thirtyDays.Timestamp
	case sixMonths := <-sixMonthsChan:
		forwardsRating.SixMonthsRating = sixMonths.Wrapper
		forwardsRating.SixMonthsTimestamp = sixMonths.Timestamp
	}
}

func calculateForwardRatingByPeriod(storage db.MetricsDatabase, metricModel *model.MetricOne,
	actualRating *RawForwardRating, acc *forwardsAccumulator, actualTimestamp int64,
	lastTimestamp int64, period time.Duration, channel chan *wrapperRawForwardRating) {

	result := NewRawForwardRating()
	timestamp := lastTimestamp
	if utime.InRangeFromUnix(actualTimestamp, lastTimestamp, period) {
		result.Success = actualRating.Success + acc.Success
		result.Failure = actualRating.Failure + acc.Failed
		result.InternalFailure = actualRating.InternalFailure + acc.LocalFailed
	} else {
		startPeriod := time.Unix(actualTimestamp, 0).Add(time.Duration(-1 * period)).Unix()
		timestamp = startPeriod
		baseID, _ := storage.ItemID(metricModel)
		startID := strings.Join([]string{baseID, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{baseID, fmt.Sprint(lastTimestamp), "metric"}, "/")
		localAcc := &forwardsAccumulator{
			Success:     0,
			Failed:      0,
			LocalFailed: 0,
		}

		err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
			if err := accumulateForwardsRating(&itemValue, localAcc); err != nil {
				log.GetInstance().Errorf("Error during counting: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.GetInstance().Errorf("During forwards rating calculation we received %s", err)
		}

		result.Success += acc.Success + localAcc.Success
		result.Failure += acc.Failed + localAcc.Failed
		result.InternalFailure += acc.LocalFailed + localAcc.LocalFailed
	}

	channel <- &wrapperRawForwardRating{
		Wrapper:   result,
		Timestamp: timestamp,
	}
}

func accumulateForwardsRating(payload *string, acc *forwardsAccumulator) error {
	var nodeMetric model.NodeMetric
	if err := json.Unmarshal([]byte(*payload), &nodeMetric); err != nil {
		return err
	}

	tmpAcc := countForwards(nodeMetric.ChannelsInfo)
	acc.Success += tmpAcc.Success
	acc.Failed += tmpAcc.Failed
	acc.LocalFailed += tmpAcc.LocalFailed
	return nil
}
