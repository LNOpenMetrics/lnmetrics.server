package metric_one

import (
	"encoding/json"
	"fmt"
	"github.com/LNOpenMetrics/lnmetrics.server/pkg/utils"
	"strings"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
)

// days constant
var TodayOccurrence = utils.ReturnOccurrence(24*time.Hour, 30*time.Minute)
var TenDaysOccurrence = utils.ReturnOccurrence(10*24*time.Hour, 30*time.Minute)
var ThirtyDaysOccurrence = utils.ReturnOccurrence(30*24*time.Hour, 30*time.Minute)
var SixMonthsOccurrence = utils.ReturnOccurrence(6*30*24*time.Hour, 30*time.Minute)

// accumulator wrapper data structure
type accumulator struct {
	Selected int64
	Total    int64
}

// Make intersection between channels information.
//
// This operation make sure that we us not outdated channels in the raw metrics
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
	err := storage.RawIterateThrough(*modelMetric.Network, startID, endID, func(itemValue string) error {
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

type wrapperUpTimeAccumulator struct {
	acc       *accumulator
	timestamp int64
}

// utils function to accumulate the UpTimeForChannels and return the accumulation value, and the last timestamp
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
			acc.Wrapper.LocalFailure++
		default:
			log.GetInstance().Errorf("Forward payment status %s unsupported", forward.Status)
		}
	}
	return acc
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
				localAcc.LocalFailure++
			}

		}
	}
	return nil
}
