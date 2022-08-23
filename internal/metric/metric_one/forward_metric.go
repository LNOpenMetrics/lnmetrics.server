package metric_one

import (
	"fmt"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
	"strings"
	"time"
)

func CalculateForwardsRatingMetricOneSync(storage db.MetricsDatabase, forwardsRating *RawForwardsRating,
	metricModel *model.MetricOne) {
	acc := CountForwardsSync(metricModel.ChannelsInfo)
	lastTimestamp := getLatestTimestamp(metricModel.UpTime)
	CalculateForwardsRatingByTimestampSync(storage, metricModel, forwardsRating, acc, lastTimestamp)
}

func CountForwardsSync(channelsInfo []*model.StatusChannel) *forwardsAccumulator {
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

type WrapperRawForwardRating struct {
	Wrapper   *RawForwardRating
	Timestamp int64
}

func NewWrapperRawForwardRating() *WrapperRawForwardRating {
	return &WrapperRawForwardRating{}
}

func CalculateForwardsRatingByTimestampSync(storage db.MetricsDatabase, metricModel *model.MetricOne,
	forwardsRating *RawForwardsRating, acc *forwardsAccumulator, actualTimestamp int64) {
	today := NewWrapperRawForwardRating()
	tenDays := NewWrapperRawForwardRating()
	thirtyDays := NewWrapperRawForwardRating()
	sixMonths := NewWrapperRawForwardRating()

	// call for today rating
	CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.TodayRating, acc,
		actualTimestamp, forwardsRating.TodayTimestamp, 1*24*time.Hour, today)

	// call for last 10 days rating
	CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.TenDaysRating, acc,
		actualTimestamp, forwardsRating.TenDaysTimestamp, 10*24*time.Hour, tenDays)

	// call for the last 30 days
	CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.ThirtyDaysTimestamp, 30*24*time.Hour, thirtyDays)

	// call for the last 6 months
	CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.SixMonthsTimestamp, 6*utime.Month, sixMonths)

	forwardsRating.TodayRating = today.Wrapper
	forwardsRating.TodayTimestamp = today.Timestamp
	forwardsRating.TenDaysRating = tenDays.Wrapper
	forwardsRating.TenDaysTimestamp = tenDays.Timestamp
	forwardsRating.ThirtyDaysRating = thirtyDays.Wrapper
	forwardsRating.ThirtyDaysTimestamp = thirtyDays.Timestamp
	forwardsRating.SixMonthsRating = sixMonths.Wrapper
	forwardsRating.SixMonthsTimestamp = sixMonths.Timestamp

	forwardsRating.FullRating.Success += acc.Success
	forwardsRating.FullRating.Failure += acc.Failed
	forwardsRating.FullRating.InternalFailure += acc.LocalFailed
}

// CalculateForwardRatingByPeriodSync Function to calculate the forward rating by period of the node that are pushing the data
//
// - storage: instance of db.MetricsDatabase that contains the wrapping around the database
// - metricModel: the metric model portion received from the node regarding the update
// - actualRating: is the actual raw rating calculate by the service
// - acc: the instance of accumulator where to make the count
// - actualTimestamp: Timestamp where the update is generated, so it is the timestamp of the MetricModel
// - lastTimestamp: Timestamp where the last metrics calculation happens, recorded in the RawModel
// - period: It is the range of period where to calculate is ran
// - channel: Is the communication channels where the communication happens
func CalculateForwardRatingByPeriodSync(storage db.MetricsDatabase, metricModel *model.MetricOne,
	actualRating *RawForwardRating, acc *forwardsAccumulator, actualTimestamp int64,
	lastTimestamp int64, period time.Duration, wrapper *WrapperRawForwardRating) {

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

	wrapper.Wrapper = result
	wrapper.Timestamp = timestamp
}