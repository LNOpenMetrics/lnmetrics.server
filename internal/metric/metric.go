package metric

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/config"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"

	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
)

type accumulator struct {
	Selected int64
	Total    int64
}

// Method to calculate the metric one output and store the result
// on the server
func CalculateMetricOneOutput(storage db.MetricsDatabase, metricModel *model.MetricOne) error {
	metricKey := strings.Join([]string{metricModel.NodeID, config.RawMetricOnePrefix}, "/")
	log.GetInstance().Infof("Raw metric for node %s key output is: %s", metricModel.NodeID, metricKey)
	rawMetric, err := storage.GetRawValue(metricKey)
	var rawMetricModel *RawMetricOneOutput
	if err != nil {
		// If the metrics it is not found, create a new one
		// and continue to fill it.
		rawMetricModel = NewRawMetricOneOutput(time.Now().Unix())
	} else {
		if err := json.Unmarshal(rawMetric, &rawMetricModel); err != nil {
			return err
		}
	}

	var lockGroup sync.WaitGroup
	lockGroup.Add(3)
	go calculateUptimeMetricOne(storage, rawMetricModel.UpTime, metricModel, &lockGroup)
	go calculateForwardsRatingMetricOne(storage, rawMetricModel.ForwardsRating, metricModel, &lockGroup)
	itemKey, _ := storage.ItemID(metricModel)
	go calculateRationForChannels(storage, itemKey, rawMetricModel.ChannelsRating, metricModel.ChannelsInfo, &lockGroup)
	lockGroup.Wait()

	rawMetricModel.LastUpdate = time.Now().Unix()
	// As result we store the value on the db
	metricModelBytes, err := json.Marshal(rawMetricModel)
	if err != nil {
		return err
	}
	log.GetInstance().Infof("Metric Calculated: %s", string(metricModelBytes))
	if err := storage.PutRawValue(metricKey, metricModelBytes); err != nil {
		return err
	}
	resKey := strings.Join([]string{metricModel.NodeID, config.MetricOneOutputSuffix}, "/")

	result := CalculateMetricOneOutputFromRaw(rawMetricModel)

	resultByte, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return storage.PutRawValue(resKey, resultByte)
}

// Execute the uptime rating of the node
// TODO: Refactoring this logic and use the channels
func calculateUptimeMetricOne(storage db.MetricsDatabase, nodeUpTime *RawPercentageData, metricModel *model.MetricOne, lock *sync.WaitGroup) {
	defer lock.Done()
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
	defer lock.Done()
	acc := countForwards(metricModel.ChannelsInfo)
	lastTimestamp := getLatestTimestamp(metricModel.UpTime)
	calculateForwardsRatingByTimestamp(storage, metricModel, forwardsRating, acc, lastTimestamp)
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

// calculate the rating for each channels in the code
//
// storage: TODO
// itemKey: TODO
// channelsRating: TODO
// channelsInfo: TODO
// lockGroud: TODO
func calculateRationForChannels(storage db.MetricsDatabase, itemKey string, channelsRating map[string]*RawChannelRating,
	channelsInfo []*model.StatusChannel, lockGroup *sync.WaitGroup) {
	defer lockGroup.Done()

	if len(channelsInfo) == 0 {
		return
	}
	chanForChannels := make(chan *RawChannelRating, len(channelsInfo)-1)
	for _, channelInfo := range channelsInfo {
		rating, found := channelsRating[channelInfo.ChannelID]
		if !found {
			sort.Slice(channelInfo.UpTime, func(i int, j int) bool { return channelInfo.UpTime[i].Timestamp < channelInfo.UpTime[j].Timestamp })
			rating = &RawChannelRating{
				Age:            int64(channelInfo.UpTime[0].Timestamp),
				ChannelID:      channelInfo.ChannelID,
				NodeID:         channelInfo.NodeID,
				Capacity:       channelInfo.Capacity,
				UpTimeRating:   NewRawPercentageData(),
				ForwardsRating: NewRawForwardsRating(),
			}
		}
		rating.Fee = channelInfo.Fee
		rating.Limits = channelInfo.Limits
		rating.Capacity = channelInfo.Capacity
		go calculateRatingForChannel(storage, itemKey, rating, channelInfo, chanForChannels)
	}

	for rating := range chanForChannels {
		channelsRating[rating.ChannelID] = rating
	}
}

// calculate the ration of one single channels
func calculateRatingForChannel(storage db.MetricsDatabase, itemKey string, channelRating *RawChannelRating,
	channelInfo *model.StatusChannel, comm chan *RawChannelRating) {
	var wg sync.WaitGroup
	wg.Add(2)

	go calculateUpTimeRatingChannel(storage, itemKey, channelInfo.ChannelID, channelRating, channelInfo.UpTime, &wg)
	go calculateForwardsPaymentsForChannel(storage, itemKey, channelInfo.ChannelID, channelRating.ForwardsRating, channelInfo.Forwards, &wg)

	wg.Wait()
}

type wrapperUpTimeAccumulator struct {
	acc       *accumulator
	timestamp int64
}

func calculateUpTimeRatingChannel(storage db.MetricsDatabase, itemKey string, channelID string, channelRating *RawChannelRating,
	upTimes []*model.ChannelStatus, lock *sync.WaitGroup) {
	defer lock.Done()

	todayChan := make(chan *wrapperUpTimeAccumulator)
	todayValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.TodaySuccess),
			Total:    int64(channelRating.UpTimeRating.TodayTotal),
		},
		timestamp: channelRating.UpTimeRating.TodayTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, todayValue, upTimes, 24*time.Hour, todayChan)
	tenDaysChan := make(chan *wrapperUpTimeAccumulator)
	tenDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.TenDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.TenDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.TenDaysTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, tenDaysValue, upTimes, 10*24*time.Hour, tenDaysChan)
	thirtyDaysChan := make(chan *wrapperUpTimeAccumulator)
	thirtyDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.ThirtyDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.ThirtyDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.ThirtyDaysTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, thirtyDaysValue, upTimes, 30*24*time.Hour, thirtyDaysChan)
	sixMonthsChan := make(chan *wrapperUpTimeAccumulator)
	sixMonthsValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.SixMonthsSuccess),
			Total:    int64(channelRating.UpTimeRating.SixMonthsTotal),
		},
		timestamp: channelRating.UpTimeRating.SixMonthsTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, sixMonthsValue, upTimes, 6*30*24*time.Hour, sixMonthsChan)

	actualValue := accumulateUpTimeForChannel(upTimes)

	channelRating.UpTimeRating.FullSuccess += uint64(actualValue.acc.Selected)
	channelRating.UpTimeRating.FullTotal += uint64(actualValue.acc.Total)

	select {
	case today := <-todayChan:
		channelRating.UpTimeRating.TodaySuccess = uint64(today.acc.Selected)
		channelRating.UpTimeRating.TodayTotal = uint64(today.acc.Total)
		channelRating.UpTimeRating.TodayTimestamp = today.timestamp
	case tenDays := <-tenDaysChan:
		channelRating.UpTimeRating.TenDaysSuccess = uint64(tenDays.acc.Selected)
		channelRating.UpTimeRating.TenDaysTotal = uint64(tenDays.acc.Total)
		channelRating.UpTimeRating.TenDaysTimestamp = tenDays.timestamp
	case thirtyDays := <-thirtyDaysChan:
		channelRating.UpTimeRating.ThirtyDaysSuccess = uint64(thirtyDays.acc.Selected)
		channelRating.UpTimeRating.ThirtyDaysTotal = uint64(thirtyDays.acc.Total)
		channelRating.UpTimeRating.ThirtyDaysTimestamp = thirtyDays.timestamp
	case sixMonths := <-sixMonthsChan:
		channelRating.UpTimeRating.SixMonthsSuccess = uint64(sixMonths.acc.Selected)
		channelRating.UpTimeRating.SixMonthsTotal = uint64(sixMonths.acc.Total)
		channelRating.UpTimeRating.SixMonthsTimestamp = sixMonths.timestamp
	}
}

// Function to abstract the function to calculate the up time in some period, in particular
// this function takes in input the following parameters:
//
// - storage: Is one implementation of the db that implement the persistence.
// - actualValues: Is the last Rating contained inside the recorded rating inside the db.
// - acc: Is the channels where the communication happens between gorutine after the calculation will finish.
// - period: Is the period where the check need to be performed
func calculateUpTimeRatingByPeriod(storage db.MetricsDatabase, itemKey string, channelID string,
	actualValues *wrapperUpTimeAccumulator, upTime []*model.ChannelStatus,
	period time.Duration, acc chan<- *wrapperUpTimeAccumulator) {
	internalAcc := accumulateUpTimeForChannel(upTime)

	if utime.InRangeFromUnix(internalAcc.timestamp, actualValues.timestamp, period) {
		internalAcc.acc.Selected += actualValues.acc.Selected
		internalAcc.acc.Total += actualValues.acc.Total
	} else {
		startPeriod := time.Unix(internalAcc.timestamp, 0).Add(time.Duration(-1 * period)).Unix()
		startID := strings.Join([]string{itemKey, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{itemKey, fmt.Sprint(internalAcc.timestamp), "metric"}, "/")
		localAcc := &accumulator{
			Selected: 0,
			Total:    0,
		}
		err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
			if err := accumulateUpTimeForChannelFromDB(channelID, &itemValue, localAcc); err != nil {
				log.GetInstance().Errorf("Error during counting: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.GetInstance().Errorf("During forwards rating calculation we received %s", err)
		}

		internalAcc.acc = localAcc
		internalAcc.timestamp = startPeriod

	}
	acc <- internalAcc

}

// utils function to accumulat the UpTimeForChannels and return the accumulation value, and the last timestamp
func accumulateUpTimeForChannel(upTime []*model.ChannelStatus) *wrapperUpTimeAccumulator {
	wrapper := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: 0,
			Total:    0,
		},
		timestamp: int64(0),
	}
	for _, item := range upTime {
		if item.Timestamp != 0 {
			wrapper.acc.Selected++
		}
		wrapper.acc.Total++
		if wrapper.timestamp < int64(item.Timestamp) {
			wrapper.timestamp = int64(item.Timestamp)
		}
	}
	return wrapper
}

func accumulateUpTimeForChannelFromDB(channelID string, payload *string, acc *accumulator) error {
	var model model.NodeMetric
	if err := json.Unmarshal([]byte(*payload), &model); err != nil {
		return err
	}

	// TODO write a json unmarshal to bind the array in a map.
	for _, channel := range model.ChannelsInfo {
		if channel.ChannelID == channelID {
			accumulateUpTime := accumulateUpTimeForChannel(channel.UpTime)
			acc.Selected += accumulateUpTime.acc.Selected
			acc.Total += accumulateUpTime.acc.Total
		}
	}

	return nil
}

// Calculate the forwards payment rating of a channel
//
// storage: TODO
// itemKey: TODO
// channelID: TODO
// forwardRating: TODO
// forwards: TODO
// wg: TODO
func calculateForwardsPaymentsForChannel(storage db.MetricsDatabase, itemKey string, channelID string,
	forwardsRating *RawForwardsRating, forwards []*model.PaymentInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	accumulation := accumulateActualForwardRating(forwards)

	todayChan := make(chan *wrapperRawForwardRating)
	tenDaysChan := make(chan *wrapperRawForwardRating)
	thirtyDaysChan := make(chan *wrapperRawForwardRating)
	sixMonthsChan := make(chan *wrapperRawForwardRating)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.TodayRating,
		accumulation, forwardsRating.TodayTimestamp, 1*24*time.Hour, todayChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.TenDaysRating,
		accumulation, forwardsRating.TenDaysTimestamp, 10*24*time.Hour, tenDaysChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.ThirtyDaysRating,
		accumulation, forwardsRating.TenDaysTimestamp, 30*24*time.Hour, thirtyDaysChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.SixMonthsRating, accumulation, forwardsRating.SixMonthsTimestamp, 6*30*24*time.Hour, sixMonthsChan)

	forwardsRating.FullRating.Success += accumulation.Wrapper.Success
	forwardsRating.FullRating.Failure += accumulation.Wrapper.Failure
	forwardsRating.FullRating.InternalFailure += accumulation.Wrapper.InternalFailure

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

func accumulateActualForwardRating(forwards []*model.PaymentInfo) *wrapperRawForwardRating {
	acc := &wrapperRawForwardRating{
		Wrapper:   NewRawForwardRating(),
		Timestamp: int64(0),
	}

	for _, forward := range forwards {
		if int64(forward.Timestamp) > acc.Timestamp {
			acc.Timestamp = int64(forward.Timestamp)
		}
		switch forward.Status {
		case "settled", "offered":
			acc.Wrapper.Success++
		case "failed":
			acc.Wrapper.Failure++
		case "local_failed":
			acc.Wrapper.InternalFailure++
		default:
			log.GetInstance().Errorf("Forward payment status %s unsupported", forward.Status)
		}
	}
	return acc
}

func accumulateForwardsRatingForChannel(storage db.MetricsDatabase, itemKey string, channelID string,
	lastForwardRating *RawForwardRating, actualForwardRating *wrapperRawForwardRating, lastTimestamp int64,
	period time.Duration, chann chan *wrapperRawForwardRating) {

	result := &wrapperRawForwardRating{
		Wrapper:   NewRawForwardRating(),
		Timestamp: lastTimestamp,
	}

	if utime.InRangeFromUnix(lastTimestamp, actualForwardRating.Timestamp, period) {
		result.Wrapper.Success = lastForwardRating.Success + actualForwardRating.Wrapper.Success
		result.Wrapper.Failure = lastForwardRating.Failure + actualForwardRating.Wrapper.Failure
		result.Wrapper.InternalFailure = lastForwardRating.InternalFailure + actualForwardRating.Wrapper.InternalFailure
	} else {
		startPeriod := time.Unix(actualForwardRating.Timestamp, 0).Add(time.Duration(-1 * period)).Unix()
		startID := strings.Join([]string{itemKey, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{itemKey, fmt.Sprint(actualForwardRating.Timestamp), "metric"}, "/")
		localAcc := NewRawForwardRating()
		err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
			if err := accumulateForwardsRatingChannelFromDB(channelID, &itemValue, localAcc); err != nil {
				log.GetInstance().Errorf("Error during counting: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.GetInstance().Errorf("During forwards rating calculation we received %s", err)
		}
		result.Wrapper.Success += localAcc.Success
		result.Wrapper.Failure += localAcc.Failure
		result.Wrapper.InternalFailure += localAcc.InternalFailure
	}

	chann <- result
}

func accumulateForwardsRatingChannelFromDB(channelID string, payload *string, localAcc *RawForwardRating) error {
	var model model.NodeMetric
	if err := json.Unmarshal([]byte(*payload), &model); err != nil {
		log.GetInstance().Errorf("Error: %s", err)
		return err
	}

	// TODO decode the list of channels info with a map to speed up the algorithm
	for _, channel := range model.ChannelsInfo {
		if channel.ChannelID != channelID {
			continue
		}
		for _, forward := range channel.Forwards {
			switch forward.Status {
			case "settled", "offered":
				localAcc.Success++
			case "failed":
				localAcc.Failure++
			case "local_failed":
				localAcc.InternalFailure++
			}

		}
	}
	return nil
}
