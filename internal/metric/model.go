package metric

// Wrapper struct that contains the information
// to calculate the metric one output in a incremental way
// we need to store a raw metric output with all the
// metadata attached.
type RawMetricOneOutput struct {
	// The version of the metric one output
	Version uint `json:"version"`
	// The last time that this raw was update
	LastUpdate uint64 `json:"last_update"`
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
	TodayTimestamp uint64 `json:"today_timestamp"`
	// Number of operation completed with success in the last 10 days
	TenDaysSuccess uint64 `json:"ten_days_success"`
	// last valid day in the range of 10 days.
	TedDaysTimestamp uint64 `json:"ten_days_timestamp"`
	// Total Number of operation in the last 10 days
	TenDatsTotal uint64 `json:"ten_days_tot"`
	// Number of operation completed with success in the last thirty days
	ThirtyDaysSuccess uint64 `json:"thirty_days_success"`
	// Total Forwards in the last 30 days
	ThirtyDaysTot uint64 `json:"thirty_days_tot"`
	// Last valid day contained in the range of the last 30 days
	ThirtyDaysTimestamp uint64 `json:"thirty_days_timestamp"`
	// Number of operation completed with success in the last 6 months
	SixMonthsSuccess uint64 `json:"six_months_success"`
	// Total Number of operation in the last 6 months
	SixMonthsTot uint64 `json:"six_months_tot"`
	// Last valid day contained in the 6 months period
	SixMonthsTimestamp uint64 `json:"six_months_timestamp"`
	// Number of operation completed with success in the all known period
	FullSuccess uint64 `json:"full_success"`
	// Total Number of operation in all the known period
	FullTot uint64 `json:"full_tot"`
}

// Wrapper struct that contains all the information about the metric one output
type RawChannelRating struct {
	UpTimeRating   *RawPercentageData `json:"up_time_rating"`
	ForwardsRating *RawForwardsRating `json:"forwards_rating"`
}

// Wrapper struct around the forwards rating
type RawForwardsRating struct {
	// Forwards Rating in the current day
	TodayRaing *RawForwardRating `json:"one_day"`
	// The timestamp of the current day
	TodayTimestamp uint64 `json:"one_day_timestamp"`
	// Forwards Rating in the last 10 days
	TenDaysRating *RawForwardRating `json:"ten_days"`
	// The timestamp of the last 10th day
	TenDaysTimestamp uint64 `json:"ten_days_timestamp"`
	// Forwards Rating of the last 30 days
	ThirtyDaysRating *RawForwardRating `json:"thirty_days"`
	// the timestamp of the 30th day
	ThirtyDaysTimestamp uint64 `json:"thirty_days_timestamp"`
	// Forwards Rating of the last 6 months
	SixMonthsRating *RawForwardRating `json:"six_months"`
	// Timestamp of the last day in the 6 month period
	SixMonthsTimestamo uint64 `json:"six_months_timestamp"`
	// Overall Forward Rating of the known period
	FullRating *RawForwardRating `json:"full"`
}

// Wrapper struct around the forward rating
// that contains information about the number
// of success, failure and internal failure
type RawForwardRating struct {
	Success uint64 `json:"success"`
	Failure uint64 `json:"failure"`
	InternalFailure uint64 `json:"internal_failure"`
}
