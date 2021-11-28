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
	ForwardsRating *RawPercentageData `json:"forwards_rating"`
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
	SullTot uint64 `json:"full_tot"`
}

// Wrapper struct that contains all the information about the metric one output
type RawChannelRating struct {
	UpTimeRating   *RawPercentageData `json:"up_time_rating"`
	ForwardsRating *RawPercentageData `json:"forwards_rating"`
}
