package metric_one

import (
	"fmt"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
	"sort"
	"strings"
	"time"
)

// CalculateRationForChannelsSync calculate the rating for each channels in the code
//
// storage: TODO
// itemKey: TODO
// channelsRating: TODO
// channelsInfo: TODO
func CalculateRationForChannelsSync(storage db.MetricsDatabase, itemKey string, channelsRating map[string]*RawChannelRating,
	channelsInfo []*model.StatusChannel) {
	if len(channelsInfo) == 0 {
		log.GetInstance().Infof("No channel information for key: %s", itemKey)
		return
	}

	for _, channelInfo := range channelsInfo {
		// Sanity check to avoid to store trash
		if channelInfo.ChannelID == "" ||
			channelInfo.Direction == "" {
			log.GetInstance().Errorf("Invalid Channels with id %s and direction %s", channelInfo.ChannelID, channelInfo.Direction)
			continue
		}

		key := strings.Join([]string{channelInfo.ChannelID, channelInfo.Direction}, "/")
		rating, found := channelsRating[key]
		if !found {
			sort.Slice(channelInfo.UpTime, func(i int, j int) bool { return channelInfo.UpTime[i].Timestamp < channelInfo.UpTime[j].Timestamp })
			validTimestamp := channelInfo.UpTime[0].Timestamp

			// FIXME: this do not make more sense?
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

		rating.Alias = channelInfo.NodeAlias
		rating.Direction = channelInfo.Direction
		rating.Fee = channelInfo.Fee
		rating.Limits = channelInfo.Limits
		rating.Capacity = channelInfo.Capacity
		CalculateRatingForChannelSync(storage, itemKey, rating, channelInfo)

		scoreKey := strings.Join([]string{rating.ChannelID, rating.Direction}, "/")
		channelsRating[scoreKey] = rating
	}
}

// CalculateRatingForChannelSync calculate the ration of one single channels
func CalculateRatingForChannelSync(storage db.MetricsDatabase, itemKey string, channelRating *RawChannelRating,
	channelInfo *model.StatusChannel) {
	CalculateUpTimeRatingChannelSync(storage, itemKey, channelInfo.ChannelID, channelRating, channelInfo.UpTime)
	CalculateForwardsPaymentsForChannelSync(storage, itemKey, channelInfo.ChannelID, channelRating.ForwardsRating, channelInfo.Forwards)
}

func CalculateUpTimeRatingChannelSync(storage db.MetricsDatabase, itemKey string,
	channelID string, channelRating *RawChannelRating, upTimes []*model.ChannelStatus) {

	todayValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.TodaySuccess),
			Total:    int64(channelRating.UpTimeRating.TodayTotal),
		},
		timestamp: channelRating.UpTimeRating.TodayTimestamp,
	}

	CalculateUpTimeRatingByPeriodSync(storage, itemKey, channelID, todayValue,
		upTimes, 24*time.Hour)

	tenDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.TenDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.TenDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.TenDaysTimestamp,
	}
	CalculateUpTimeRatingByPeriodSync(storage, itemKey, channelID, tenDaysValue,
		upTimes, 10*24*time.Hour)

	thirtyDaysValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.ThirtyDaysSuccess),
			Total:    int64(channelRating.UpTimeRating.ThirtyDaysTotal),
		},
		timestamp: channelRating.UpTimeRating.ThirtyDaysTimestamp,
	}
	CalculateUpTimeRatingByPeriodSync(storage, itemKey, channelID, thirtyDaysValue,
		upTimes, 30*24*time.Hour)

	sixMonthsValue := &wrapperUpTimeAccumulator{
		acc: &accumulator{
			Selected: int64(channelRating.UpTimeRating.SixMonthsSuccess),
			Total:    int64(channelRating.UpTimeRating.SixMonthsTotal),
		},
		timestamp: channelRating.UpTimeRating.SixMonthsTimestamp,
	}
	CalculateUpTimeRatingByPeriodSync(storage, itemKey, channelID, sixMonthsValue,
		upTimes, 6*30*24*time.Hour)

	actualValue := accumulateUpTimeForChannel(upTimes)
	lastTimestamp := int64(0)
	for _, upTime := range upTimes {
		if int64(upTime.Timestamp) > lastTimestamp {
			lastTimestamp = int64(upTime.Timestamp)
		}
	}

	channelRating.UpTimeRating.TodaySuccess = uint64(todayValue.acc.Selected)
	channelRating.UpTimeRating.TodayTotal = TodayOccurrence
	channelRating.UpTimeRating.TodayTimestamp = todayValue.timestamp

	channelRating.UpTimeRating.TenDaysSuccess = uint64(tenDaysValue.acc.Selected)
	channelRating.UpTimeRating.TenDaysTotal = RenDaysOccurrence
	channelRating.UpTimeRating.TenDaysTimestamp = tenDaysValue.timestamp

	channelRating.UpTimeRating.ThirtyDaysSuccess = uint64(thirtyDaysValue.acc.Selected)
	channelRating.UpTimeRating.ThirtyDaysTotal = ThirtyDaysOccurrence
	channelRating.UpTimeRating.ThirtyDaysTimestamp = thirtyDaysValue.timestamp

	channelRating.UpTimeRating.SixMonthsSuccess = uint64(sixMonthsValue.acc.Selected)
	channelRating.UpTimeRating.SixMonthsTotal = SixMonthsOccurrence
	channelRating.UpTimeRating.SixMonthsTimestamp = sixMonthsValue.timestamp

	channelRating.UpTimeRating.FullSuccess += uint64(actualValue.acc.Selected)
	channelRating.UpTimeRating.FullTotal = utime.OccurenceInUnixRange(channelRating.Age, lastTimestamp, 30*time.Minute)
}

// CalculateUpTimeRatingByPeriodSync Function to abstract the function to calculate the up time in some period, in particular
// this function takes in input the following parameters:
//
// - storage: Is one implementation of the db that implement the persistence.
// - actualValues: Is the last Rating contained inside the recorded rating inside the db.
// - acc: Is the channels where the communication happens between gorutine after the calculation will finish.
// - period: Is the period where the check need to be performed
func CalculateUpTimeRatingByPeriodSync(storage db.MetricsDatabase, itemKey string, channelID string,
	actualValues *wrapperUpTimeAccumulator, upTime []*model.ChannelStatus,
	period time.Duration) {
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
}

// CalculateForwardsPaymentsForChannelSync Calculate the forwards payment rating of a channel
//
// storage: TODO
// itemKey: TODO
// channelID: TODO
// forwardRating: TODO
// forwards: TODO
// wg: TODO
func CalculateForwardsPaymentsForChannelSync(storage db.MetricsDatabase, itemKey string, channelID string,
	forwardsRating *RawForwardsRating, forwards []*model.PaymentInfo) {

	today := accumulateForwardsRatingForChannelSync(storage, itemKey, channelID, forwardsRating.TodayRating,
		forwards, forwardsRating.TodayTimestamp, 1*24*time.Hour)

	tenDays := accumulateForwardsRatingForChannelSync(storage, itemKey, channelID, forwardsRating.TenDaysRating,
		forwards, forwardsRating.TenDaysTimestamp, 10*24*time.Hour)

	thirtyDays := accumulateForwardsRatingForChannelSync(storage, itemKey, channelID, forwardsRating.ThirtyDaysRating,
		forwards, forwardsRating.ThirtyDaysTimestamp, 30*24*time.Hour)

	sixMonths := accumulateForwardsRatingForChannelSync(storage, itemKey, channelID, forwardsRating.SixMonthsRating,
		forwards, forwardsRating.SixMonthsTimestamp, 6*30*24*time.Hour)

	forwardsRating.TodayRating = today.Wrapper
	forwardsRating.TodayTimestamp = today.Timestamp

	forwardsRating.TenDaysRating = tenDays.Wrapper
	forwardsRating.TenDaysTimestamp = tenDays.Timestamp

	forwardsRating.ThirtyDaysRating = thirtyDays.Wrapper
	forwardsRating.ThirtyDaysTimestamp = thirtyDays.Timestamp

	forwardsRating.SixMonthsRating = sixMonths.Wrapper
	forwardsRating.SixMonthsTimestamp = sixMonths.Timestamp

	accumulation := accumulateActualForwardRating(forwards)
	forwardsRating.FullRating.Success += accumulation.Wrapper.Success
	forwardsRating.FullRating.Failure += accumulation.Wrapper.Failure
	forwardsRating.FullRating.InternalFailure += accumulation.Wrapper.InternalFailure
}

func accumulateForwardsRatingForChannelSync(storage db.MetricsDatabase, itemKey string, channelID string,
	lastForwardRating *RawForwardRating, forwards []*model.PaymentInfo, lastTimestamp int64,
	period time.Duration) *wrapperRawForwardRating {

	result := accumulateActualForwardRating(forwards)

	if utime.InRangeFromUnix(result.Timestamp, lastTimestamp, period) {
		result.Wrapper.Success += lastForwardRating.Success
		result.Wrapper.Failure += lastForwardRating.Failure
		result.Wrapper.InternalFailure += lastForwardRating.InternalFailure
	} else {
		startPeriod := utime.SubToTimestamp(result.Timestamp, period)
		startID := strings.Join([]string{itemKey, fmt.Sprint(startPeriod), "metric"}, "/")
		endID := strings.Join([]string{itemKey, fmt.Sprint(result.Timestamp + 1), "metric"}, "/")
		err := storage.RawIterateThrough(startID, endID, func(itemValue string) error {
			if err := accumulateForwardsRatingChannelFromDB(channelID, &itemValue, result.Wrapper); err != nil {
				log.GetInstance().Errorf("Error during counting: %s", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.GetInstance().Errorf("During forwards rating calculation we received %s", err)
		}
	}
	return result
}
