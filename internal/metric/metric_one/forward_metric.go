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
	// Timestamp where the metrics is calculated
	// we use the server timestamp because the metrics timestamp can be
	// not reliable (can be 0).
	actualTimestamp := time.Now().Unix()
	CalculateForwardsRatingByTimestampSync(storage, metricModel, forwardsRating, acc, actualTimestamp)
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

func CalculateForwardsRatingByTimestampSync(storage db.MetricsDatabase, metricModel *model.MetricOne,
	forwardsRating *RawForwardsRating, acc *forwardsAccumulator, actualTimestamp int64) {

	// call for today rating
	today := CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.TodayRating, acc,
		actualTimestamp, forwardsRating.TodayTimestamp, 1*24*time.Hour)

	// call for last 10 days rating
	tenDays := CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.TenDaysRating, acc,
		actualTimestamp, forwardsRating.TenDaysTimestamp, 10*24*time.Hour)

	// call for the last 30 days
	thirtyDays := CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.ThirtyDaysTimestamp, 30*24*time.Hour)

	// call for the last 6 months
	sixMonths := CalculateForwardRatingByPeriodSync(storage, metricModel, forwardsRating.ThirtyDaysRating, acc,
		actualTimestamp, forwardsRating.SixMonthsTimestamp, 6*utime.Month)

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
	forwardsRating.FullRating.LocalFailure += acc.LocalFailed
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
	lastTimestamp int64, period time.Duration) *WrapperRawForwardRating {

	result := NewRawForwardRating()
	timestamp := lastTimestamp
	if utime.InRangeFromUnix(actualTimestamp, lastTimestamp, period) {
		result.Success = actualRating.Success + acc.Success
		result.Failure = actualRating.Failure + acc.Failed
		result.InternalFailure = actualRating.InternalFailure + acc.LocalFailed
		result.LocalFailure = actualRating.LocalFailure + actualRating.LocalFailure
	} else {
		timestamp = utime.SubToTimestamp(actualTimestamp, period)
		baseID, _ := storage.ItemID(metricModel)
		startID := strings.Join([]string{baseID, fmt.Sprint(timestamp), "metric"}, "/")
		endID := strings.Join([]string{baseID, fmt.Sprint(actualTimestamp + 1), "metric"}, "/")
		localAcc := &forwardsAccumulator{
			Success:     0,
			Failed:      0,
			LocalFailed: 0,
		}

		err := storage.RawIterateThrough(*metricModel.Network, startID, endID, func(itemValue string) error {
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
		result.LocalFailure = acc.LocalFailed + localAcc.LocalFailed
	}
	return &WrapperRawForwardRating{
		Wrapper:   result,
		Timestamp: timestamp,
	}
}
