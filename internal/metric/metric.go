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

// days constant

var todayOccurence = returnOccurence(24*time.Hour, 30*time.Minute)
var tenDaysOccurence = returnOccurence(10*24*time.Hour, 30*time.Minute)
var thirtyDaysOccurence = returnOccurence(30*24*time.Hour, 30*time.Minute)
var sixMonthsOccurence = returnOccurence(6*30*24*time.Hour, 30*time.Minute)

// accumulator wrapper data structure
type accumulator struct {
	Selected int64
	Total    int64
}

// Make intersection between channels information.
//
// This operation make sure that we us not outdate channels in the raw metrics
// but only channels that are open right now on the node.
//
// updateState: The new State received from the node
// oldState: the State of the metric that it is stored in the database
func intersectionChannelsInfo(updateState *model.MetricOne, oldState *RawMetricOneOutput) error {
	for _, channel := range updateState.ChannelsInfo {
		key := strings.Join([]string{channel.ChannelID, channel.Direction}, "/")
		ratingChannel, found := oldState.ChannelsRating[key]
		if !found {
			delete(oldState.ChannelsRating, key)
		} else {
			if ratingChannel.Alias == "" && ratingChannel.Direction == "" {
				delete(oldState.ChannelsRating, key)
			}
		}
	}
	return nil
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

	// Make intersection between channels info
	// this give the possibility to remove from the raw metrics
	// channels that are not longer available
	if err := intersectionChannelsInfo(metricModel, rawMetricModel); err != nil {
		log.GetInstance().Errorf("Error: %s", err)
		return nil
	}

	var lockGroup sync.WaitGroup
	lockGroup.Add(3)
	go calculateUptimeMetricOne(storage, rawMetricModel, metricModel, &lockGroup)
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

func returnOccurence(period time.Duration, step time.Duration) uint64 {
	now := time.Now()
	after := now.Add(period)
	return utime.OccurenceInUnixRange(now.Unix(), after.Unix(), step)
}

// Execute the uptime rating of the node
// TODO: Refactoring this logic and use the channels
func calculateUptimeMetricOne(storage db.MetricsDatabase, rawMetric *RawMetricOneOutput,
	metricModel *model.MetricOne, lock *sync.WaitGroup) {
	defer lock.Done()
	onlineUpdate := uint64(0)
	lastTimestamp := int64(-1)
	for _, upTime := range metricModel.UpTime {
		// We take into count only the on_update that are generated by the
		// plugin timeout.
		// we could make this calculation better to take in account also the
		// previous event, but for now we implement the easy things
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
		// reset
		// This is a correct assumtion? or I need to iterate inside the DB?
		nodeUpTime.TodaySuccess = onlineUpdate
	}
	nodeUpTime.TodayTotal = todayOccurence
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
	nodeUpTime.TenDaysTotal = tenDaysOccurence

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
	nodeUpTime.ThirtyDaysTotal = thirtyDaysOccurence

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
	nodeUpTime.SixMonthsTotal = sixMonthsOccurence

	// full
	nodeUpTime.FullSuccess += onlineUpdate
	nodeUpTime.FullTotal = utime.OccurenceInUnixRange(rawMetric.Age, lastTimestamp, 30*time.Minute)
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
func walkThroughUpTime(storage db.MetricsDatabase, modelMetric *model.MetricOne,
	startDate int64, endDate int64, acc *accumulator) error {
	baseID, _ := storage.ItemID(modelMetric)
	startID := strings.Join([]string{baseID, fmt.Sprint(startDate), "metric"}, "/")
	endID := strings.Join([]string{baseID, fmt.Sprint(endDate + 1), "metric"}, "/")
	err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
		if err := accumulateUpTime(itemValue, acc); err != nil {
			log.GetInstance().Errorf("Error during counting: %s", err)
			return err
		}
		return nil
	})
	return err
}

// Function wrapper to accumulate the forwards payments counter
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

	for i := 0; i < 4; i++ {
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
}

// Function to calculate the forward rating by period of the node that are pushing the data
//
// - storage: a instance of db.MetricsDatabase that contains the wrapping around the database
// - metricModel: the metric model portion received from the node regarding the update
// - actualRating: is the actual raw rating calculate by the service
// - acc: the instance of accumulator where to make the count
// - actualTimestamp: Timestamp where the update is generated, so it is the timestamp of the MetricModel
// - lastTimestamp: Timestamp where the last metrics calculation happens, recorded in the RawModel
// - period: It is the range of period where the calculate is ran
// - channel: Is the communication channels where the communication happens
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
		startPeriod := utime.SubToTimestamp(actualTimestamp, period)
		baseID, _ := storage.ItemID(metricModel)
		startID := strings.Join([]string{baseID, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{baseID, fmt.Sprint(actualTimestamp + 1), "metric"}, "/")
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

		result.Success = acc.Success + localAcc.Success
		result.Failure = acc.Failed + localAcc.Failed
		result.InternalFailure = acc.LocalFailed + localAcc.LocalFailed
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
		if channelInfo.ChannelID == "" ||
			channelInfo.Direction == "" {
			log.GetInstance().Errorf("Invalid Channels with id %s and direction %s", channelInfo.ChannelID, channelInfo.Direction)
			continue
		}
		// TODO: We need to aggregate the channels only one, or we need to
		// keep in and out channel? The second motivation sound good to method
		key := strings.Join([]string{channelInfo.ChannelID, channelInfo.Direction}, "/")
		rating, found := channelsRating[key]
		if !found {
			sort.Slice(channelInfo.UpTime, func(i int, j int) bool { return channelInfo.UpTime[i].Timestamp < channelInfo.UpTime[j].Timestamp })
			validTimestamp := channelInfo.UpTime[0].Timestamp
			if validTimestamp <= 0 {
				for _, upTime := range channelInfo.UpTime {
					if upTime.Timestamp > 0 {
						validTimestamp = upTime.Timestamp
					}
				}
			}
			rating = &RawChannelRating{
				Age:            int64(validTimestamp),
				ChannelID:      channelInfo.ChannelID,
				NodeID:         channelInfo.NodeID,
				Alias:          channelInfo.NodeAlias,
				Direction:      channelInfo.Direction,
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

	for i := 0; i < len(channelsInfo); i++ {
		rating := <-chanForChannels
		key := strings.Join([]string{rating.ChannelID, rating.Direction}, "/")
		channelsRating[key] = rating
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
	comm <- channelRating
}

type wrapperUpTimeAccumulator struct {
	acc       *accumulator
	timestamp int64
}

func calculateUpTimeRatingChannel(storage db.MetricsDatabase, itemKey string,
	channelID string, channelRating *RawChannelRating,
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
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, todayValue,
		upTimes, 24*time.Hour, todayChan)

	tenDaysChan := make(chan *wrapperUpTimeAccumulator)
	tenDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.TenDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.TenDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.TenDaysTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, tenDaysValue,
		upTimes, 10*24*time.Hour, tenDaysChan)

	thirtyDaysChan := make(chan *wrapperUpTimeAccumulator)
	thirtyDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.ThirtyDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.ThirtyDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.ThirtyDaysTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, thirtyDaysValue,
		upTimes, 30*24*time.Hour, thirtyDaysChan)

	sixMonthsChan := make(chan *wrapperUpTimeAccumulator)
	sixMonthsValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.SixMonthsSuccess),
			Total:    int64(channelRating.UpTimeRating.SixMonthsTotal),
		},
		timestamp: channelRating.UpTimeRating.SixMonthsTimestamp,
	}
	go calculateUpTimeRatingByPeriod(storage, itemKey, channelID, sixMonthsValue,
		upTimes, 6*30*24*time.Hour, sixMonthsChan)

	actualValue := accumulateUpTimeForChannel(upTimes)
	lastTimestamp := int64(0)
	for _, upTime := range upTimes {
		if int64(upTime.Timestamp) > lastTimestamp {
			lastTimestamp = int64(upTime.Timestamp)
		}
	}

	channelRating.UpTimeRating.FullSuccess += uint64(actualValue.acc.Selected)
	channelRating.UpTimeRating.FullTotal = utime.OccurenceInUnixRange(channelRating.Age, lastTimestamp, 30*time.Minute)

	for i := 0; i < 4; i++ {
		select {
		case today := <-todayChan:
			channelRating.UpTimeRating.TodaySuccess = uint64(today.acc.Selected)
			channelRating.UpTimeRating.TodayTotal = todayOccurence
			channelRating.UpTimeRating.TodayTimestamp = today.timestamp
		case tenDays := <-tenDaysChan:
			channelRating.UpTimeRating.TenDaysSuccess = uint64(tenDays.acc.Selected)
			channelRating.UpTimeRating.TenDaysTotal = tenDaysOccurence
			channelRating.UpTimeRating.TenDaysTimestamp = tenDays.timestamp
		case thirtyDays := <-thirtyDaysChan:
			channelRating.UpTimeRating.ThirtyDaysSuccess = uint64(thirtyDays.acc.Selected)
			channelRating.UpTimeRating.ThirtyDaysTotal = thirtyDaysOccurence
			channelRating.UpTimeRating.ThirtyDaysTimestamp = thirtyDays.timestamp
		case sixMonths := <-sixMonthsChan:
			channelRating.UpTimeRating.SixMonthsSuccess = uint64(sixMonths.acc.Selected)
			channelRating.UpTimeRating.SixMonthsTotal = sixMonthsOccurence
			channelRating.UpTimeRating.SixMonthsTimestamp = sixMonths.timestamp
		}
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
		startPeriod := utime.SubToTimestamp(internalAcc.timestamp, period)
		startID := strings.Join([]string{itemKey, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{itemKey, fmt.Sprint(internalAcc.timestamp + 1), "metric"}, "/")
		localAcc := &accumulator{
			Selected: internalAcc.acc.Selected,
			Total:    internalAcc.acc.Total,
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
			Total:    int64(len(upTime)),
		},
		timestamp: int64(-1),
	}
	for _, item := range upTime {
		// The event take into count is the on_update because
		// it is the only event type originated by the plugin
		// and not triggered by the user.
		// event like on_start, on_close are not important to calculate
		// the up time of the node channel.
		if item.Timestamp != 0 && item.Event == "on_update" {
			wrapper.acc.Selected++
		}
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

	todayChan := make(chan *wrapperRawForwardRating)
	tenDaysChan := make(chan *wrapperRawForwardRating)
	thirtyDaysChan := make(chan *wrapperRawForwardRating)
	sixMonthsChan := make(chan *wrapperRawForwardRating)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.TodayRating,
		forwards, forwardsRating.TodayTimestamp, 1*24*time.Hour, todayChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.TenDaysRating,
		forwards, forwardsRating.TenDaysTimestamp, 10*24*time.Hour, tenDaysChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.ThirtyDaysRating,
		forwards, forwardsRating.ThirtyDaysTimestamp, 30*24*time.Hour, thirtyDaysChan)

	go accumulateForwardsRatingForChannel(storage, itemKey, channelID, forwardsRating.SixMonthsRating,
		forwards, forwardsRating.SixMonthsTimestamp, 6*30*24*time.Hour, sixMonthsChan)

	accumulation := accumulateActualForwardRating(forwards)
	forwardsRating.FullRating.Success += accumulation.Wrapper.Success
	forwardsRating.FullRating.Failure += accumulation.Wrapper.Failure
	forwardsRating.FullRating.InternalFailure += accumulation.Wrapper.InternalFailure

	for i := 0; i < 4; i++ {
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
	lastForwardRating *RawForwardRating, forwards []*model.PaymentInfo, lastTimestamp int64,
	period time.Duration, chann chan *wrapperRawForwardRating) {

	result := accumulateActualForwardRating(forwards)

	if utime.InRangeFromUnix(result.Timestamp, lastTimestamp, period) {
		result.Wrapper.Success += lastForwardRating.Success
		result.Wrapper.Failure += lastForwardRating.Failure
		result.Wrapper.InternalFailure += lastForwardRating.InternalFailure
	} else {
		startPeriod := utime.SubToTimestamp(result.Timestamp, period)
		startID := strings.Join([]string{itemKey, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{itemKey, fmt.Sprint(result.Timestamp + 1), "metric"}, "/")
		acc := result.Wrapper
		err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
			if err := accumulateForwardsRatingChannelFromDB(channelID, &itemValue, acc); err != nil {
				log.GetInstance().Errorf("Error during counting: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.GetInstance().Errorf("During forwards rating calculation we received %s", err)
		}
	}

	chann <- result
}

func accumulateForwardsRatingChannelFromDB(channelID string, payload *string, localAcc *RawForwardRating) error {
	var model model.NodeMetric
	if err := json.Unmarshal([]byte(*payload), &model); err != nil {
		log.GetInstance().Errorf("Error: %s", err)
		return err
	}

	// FIXME: We can decode the list of channels info in a map to speed up the algorithm
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
