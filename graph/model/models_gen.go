// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type ChannelFee struct {
	Base    int `json:"base"`
	PerMSat int `json:"per_msat"`
}

type ChannelInfoOutput struct {
	Age            int                    `json:"age"`
	ChannelID      string                 `json:"channel_id"`
	Alias          string                 `json:"alias"`
	Direction      string                 `json:"direction"`
	NodeID         string                 `json:"node_id"`
	Capacity       int                    `json:"capacity"`
	Fee            *ChannelFee            `json:"fee"`
	Limits         *ChannelLimits         `json:"limits"`
	UpTime         *UpTimeOutput          `json:"up_time"`
	ForwardsRating *ForwardsRatingSummary `json:"forwards_rating"`
}

type ChannelLimits struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type ChannelStatus struct {
	Event     string `json:"event"`
	Timestamp int    `json:"timestamp"`
	Status    string `json:"status"`
}

type ChannelSummary struct {
	NodeID    string `json:"node_id"`
	Alias     string `json:"alias"`
	Color     string `json:"color"`
	ChannelID string `json:"channel_id"`
	State     string `json:"state"`
}

type ChannelsSummary struct {
	TotChannels int               `json:"tot_channels"`
	Summary     []*ChannelSummary `json:"summary"`
}

type ForwardsRating struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	// Deprecated
	InternalFailure int `json:"internal_failure"`
	LocalFailure    int `json:"local_failure"`
}

type ForwardsRatingSummary struct {
	OneDay     *ForwardsRating `json:"one_day"`
	TenDays    *ForwardsRating `json:"ten_days"`
	ThirtyDays *ForwardsRating `json:"thirty_days"`
	SixMonths  *ForwardsRating `json:"six_months"`
	Full       *ForwardsRating `json:"full"`
}

type MetricOne struct {
	Name         string           `json:"metric_name"`
	NodeID       string           `json:"node_id"`
	Color        string           `json:"color"`
	NodeAlias    string           `json:"node_alias"`
	Network      *string          `json:"network"`
	OSInfo       *OSInfo          `json:"os_info"`
	NodeInfo     *NodeImpInfo     `json:"node_info"`
	Address      []*NodeAddress   `json:"address"`
	Timezone     string           `json:"timezone"`
	UpTime       []*Status        `json:"up_time"`
	ChannelsInfo []*StatusChannel `json:"channels_info"`
	Version      *int             `json:"version"`
}

type MetricOneInfo struct {
	UpTime       []*Status        `json:"up_time"`
	ChannelsInfo []*StatusChannel `json:"channels_info"`
	PageInfo     *PageInfo        `json:"page_info"`
}

type MetricOneOutput struct {
	Version        int                    `json:"version"`
	Age            int                    `json:"age"`
	LastUpdate     int                    `json:"last_update"`
	ForwardsRating *ForwardsRatingSummary `json:"forwards_rating"`
	UpTime         *UpTimeOutput          `json:"up_time"`
	ChannelsInfo   []*ChannelInfoOutput   `json:"channels_info"`
}

type NodeAddress struct {
	Type string `json:"type"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type NodeImpInfo struct {
	Implementation string `json:"implementation"`
	Version        string `json:"version"`
}

type NodeInfo struct {
	NodeID    string     `json:"node_id"`
	MetricOne *MetricOne `json:"metric_one"`
}

type NodeMetadata struct {
	Version    int            `json:"version"`
	NodeID     string         `json:"node_id"`
	Alias      string         `json:"alias"`
	Color      string         `json:"color"`
	Address    []*NodeAddress `json:"address"`
	Network    string         `json:"network"`
	OSInfo     *OSInfo        `json:"os_info"`
	NodeInfo   *NodeImpInfo   `json:"node_info"`
	Timezone   string         `json:"timezone"`
	LastUpdate int            `json:"last_update"`
}

type NodeMetric struct {
	Timestamp    int              `json:"timestamp"`
	UpTime       []*Status        `json:"up_time"`
	ChannelsInfo []*StatusChannel `json:"channels_info"`
}

type OSInfo struct {
	Os           string `json:"os"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
}

type PageInfo struct {
	StartCursor int  `json:"start"`
	EndCursor   int  `json:"end"`
	HasNext     bool `json:"has_next"`
}

type PaymentInfo struct {
	Direction     string `json:"direction"`
	Status        string `json:"status"`
	FailureReason string `json:"failure_reason"`
	FailureCode   int    `json:"failure_code"`
	Timestamp     int    `json:"timestamp"`
}

type PaymentsSummary struct {
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

type Status struct {
	Event     string           `json:"event"`
	Channels  *ChannelsSummary `json:"channels"`
	Forwards  *PaymentsSummary `json:"forwards"`
	Timestamp int              `json:"timestamp"`
	Fee       *ChannelFee      `json:"fee"`
	Limits    *ChannelLimits   `json:"limits"`
}

type StatusChannel struct {
	ChannelID  string           `json:"channel_id"`
	NodeID     string           `json:"node_id"`
	NodeAlias  string           `json:"node_alias"`
	Color      string           `json:"color"`
	Capacity   int              `json:"capacity"`
	Forwards   []*PaymentInfo   `json:"forwards"`
	UpTime     []*ChannelStatus `json:"up_time"`
	Online     bool             `json:"online"`
	LastUpdate int              `json:"last_update"`
	Direction  string           `json:"direction"`
	Fee        *ChannelFee      `json:"fee"`
	Limits     *ChannelLimits   `json:"limits"`
}

type UpTimeOutput struct {
	OneDay     int `json:"one_day"`
	TenDays    int `json:"ten_days"`
	ThirtyDays int `json:"thirty_days"`
	SixMonths  int `json:"six_months"`
	Full       int `json:"full"`
}
