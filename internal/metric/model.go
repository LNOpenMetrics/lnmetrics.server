package metric

import (
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
)

// Wrapper struct that contains the information
// to calculate the metric one output in a incremental way
// we need to store a raw metric output with all the
// metadata attached.
type RawMetricOneOutput struct {
	// The version of the metric one output
	Version uint `json:"version"`
	// The last time that this raw was update
	LastUpdate int64 `json:"last_update"`
	//forwards rating with all the information attached
	ForwardsRating *RawForwardsRating `json:"forwards_rating"`
	// UpTime Rating about the node
	UpTime *RawPercentageData `json:"up_time"`
	// Channels rating that give us the possibility to calculate
	// the rating easily
	ChannelsRating map[string]*RawChannelRating `json:"channels_rating"`
}

// Wrapper struct that contains all the information to
// calculate the percentage
type RawPercentageData struct {
	// Number of operation with success in the last day
	TodaySuccess uint64 `json:"today_success"`
	// Tot number of operation with success in the last day
	TodayTotal uint64 `json:"today_tot"`
	// Last day timestamp to check if this data collected are
	// still valid
	TodayTimestamp int64 `json:"today_timestamp"`
	// Number of operation completed with success in the last 10 days
	TenDaysSuccess uint64 `json:"ten_days_success"`
	// last valid day in the range of 10 days.
	TenDaysTimestamp int64 `json:"ten_days_timestamp"`
	// Total Number of operation in the last 10 days
	TenDaysTotal uint64 `json:"ten_days_tot"`
	// Number of operation completed with success in the last thirty days
	ThirtyDaysSuccess uint64 `json:"thirty_days_success"`
	// Total Forwards in the last 30 days
	ThirtyDaysTotal uint64 `json:"thirty_days_tot"`
	// Last valid day contained in the range of the last 30 days
	ThirtyDaysTimestamp int64 `json:"thirty_days_timestamp"`
	// Number of operation completed with success in the last 6 months
	SixMonthsSuccess uint64 `json:"six_months_success"`
	// Total Number of operation in the last 6 months
	SixMonthsTotal uint64 `json:"six_months_tot"`
	// Last valid day contained in the 6 months period
	SixMonthsTimestamp int64 `json:"six_months_timestamp"`
	// Number of operation completed with success in the all known period
	FullSuccess uint64 `json:"full_success"`
	// Total Number of operation in all the known period
	FullTotal uint64 `json:"full_tot"`
}

// Wrapper struct that contains all the information about the metric one output
type RawChannelRating struct {
	// Age of the channels, it is the first time that the node
	// is catch from the channel
	Age            int64                `json:"age"`
	ChannelID      string               `json:"channel_id"`
	NodeID         string               `json:"node_id"`
	Capacity       int64                `json:"capacity"`
	Fee            *model.ChannelFee    `json:"fee"`
	Limits         *model.ChannelLimits `json:"limits"`
	UpTimeRating   *RawPercentageData   `json:"up_time_rating"`
	ForwardsRating *RawForwardsRating   `json:"forwards_rating"`
}

// Wrapper struct around the forwards rating
type RawForwardsRating struct {
	// Forwards Rating in the current day
	TodayRating *RawForwardRating `json:"one_day"`
	// The timestamp of the current day
	TodayTimestamp int64 `json:"one_day_timestamp"`
	// Forwards Rating in the last 10 days
	TenDaysRating *RawForwardRating `json:"ten_days"`
	// The timestamp of the last 10th day
	TenDaysTimestamp int64 `json:"ten_days_timestamp"`
	// Forwards Rating of the last 30 days
	ThirtyDaysRating *RawForwardRating `json:"thirty_days"`
	// the timestamp of the 30th day
	ThirtyDaysTimestamp int64 `json:"thirty_days_timestamp"`
	// Forwards Rating of the last 6 months
	SixMonthsRating *RawForwardRating `json:"six_months"`
	// Timestamp of the last day in the 6 month period
	SixMonthsTimestamp int64 `json:"six_months_timestamp"`
	// Overall Forward Rating of the known period
	FullRating *RawForwardRating `json:"full"`
}

// Wrapper struct around the forward rating
// that contains information about the number
// of success, failure and internal failure
type RawForwardRating struct {
	Success         uint64 `json:"success"`
	Failure         uint64 `json:"failure"`
	InternalFailure uint64 `json:"internal_failure"`
}

// Create a new Raw Metric One Output with all the default value
func NewRawMetricOneOutput(timestamp int64) *RawMetricOneOutput {
	return &RawMetricOneOutput{
		Version:        0,
		LastUpdate:     timestamp,
		ForwardsRating: NewRawForwardsRating(),
		UpTime:         NewRawPercentageData(),
		ChannelsRating: make(map[string]*RawChannelRating),
	}
}

func NewRawPercentageData() *RawPercentageData {
	return &RawPercentageData{
		TodaySuccess:        0,
		TodayTotal:          0,
		TodayTimestamp:      0,
		TenDaysSuccess:      0,
		TenDaysTimestamp:    0,
		TenDaysTotal:        0,
		ThirtyDaysSuccess:   0,
		ThirtyDaysTotal:     0,
		ThirtyDaysTimestamp: 0,
		SixMonthsSuccess:    0,
		SixMonthsTotal:      0,
		SixMonthsTimestamp:  0,
		FullSuccess:         0,
		FullTotal:           0,
	}
}

func NewRawForwardsRating() *RawForwardsRating {
	return &RawForwardsRating{
		TodayRating:         NewRawForwardRating(),
		TodayTimestamp:      time.Now().Unix(),
		TenDaysRating:       NewRawForwardRating(),
		TenDaysTimestamp:    time.Now().Unix(),
		ThirtyDaysRating:    NewRawForwardRating(),
		ThirtyDaysTimestamp: time.Now().Unix(),
		SixMonthsRating:     NewRawForwardRating(),
		SixMonthsTimestamp:  time.Now().Unix(),
		FullRating:          NewRawForwardRating(),
	}
}

// Create a new item that contains information about the forwards status.
func NewRawForwardRating() *RawForwardRating {
	return &RawForwardRating{
		Success:         0,
		Failure:         0,
		InternalFailure: 0,
	}
}
