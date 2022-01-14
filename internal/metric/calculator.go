package metric

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
)

// TODO move inside the utils function
func Percentage(selected uint64, total uint64) int {
	if total == 0 {
		return 0
	}
	return int((float64(selected)/float64(total))*100) % 100
}

func CalculateMetricOneOutputFromRaw(rawMetric *RawMetricOneOutput) *model.MetricOneOutput {
	result := &model.MetricOneOutput{}

	result.Version = int(rawMetric.Version)
	result.Age = int(rawMetric.Age)
	result.LastUpdate = int(rawMetric.LastUpdate)

	result.ForwardsRating = MappingForwardsRatingFromRaw(rawMetric.ForwardsRating)

	result.UpTime = MappingUpTimeRatingFromRaw(rawMetric.UpTime)
	result.ChannelsInfo = MappingChannelsInfoFromRaw(rawMetric.ChannelsRating)
	return result
}

func MappingForwardsRatingFromRaw(raw *RawForwardsRating) *model.ForwardsRatingSummary {
	return &model.ForwardsRatingSummary{
		OneDay:     MappingForwardsRatingByDayFromRaw(raw.TodayRating),
		TenDays:    MappingForwardsRatingByDayFromRaw(raw.TenDaysRating),
		ThirtyDays: MappingForwardsRatingByDayFromRaw(raw.ThirtyDaysRating),
		SixMonths:  MappingForwardsRatingByDayFromRaw(raw.SixMonthsRating),
		Full:       MappingForwardsRatingByDayFromRaw(raw.FullRating),
	}
}

func MappingForwardsRatingByDayFromRaw(raw *RawForwardRating) *model.ForwardsRating {
	tot := raw.Success + raw.Failure + raw.InternalFailure
	return &model.ForwardsRating{
		Success:         Percentage(raw.Success, tot),
		Failure:         Percentage(raw.Failure, tot),
		InternalFailure: Percentage(raw.InternalFailure, tot),
	}
}

func MappingUpTimeRatingFromRaw(raw *RawPercentageData) *model.UpTimeOutput {
	return &model.UpTimeOutput{
		OneDay:     Percentage(raw.TodaySuccess, raw.TodayTotal),
		TenDays:    Percentage(raw.TenDaysSuccess, raw.TenDaysTotal),
		ThirtyDays: Percentage(raw.ThirtyDaysSuccess, raw.ThirtyDaysTotal),
		SixMonths:  Percentage(raw.SixMonthsSuccess, raw.SixMonthsTotal),
		Full:       Percentage(raw.FullSuccess, raw.FullTotal),
	}
}

func MappingChannelsInfoFromRaw(raw map[string]*RawChannelRating) []*model.ChannelInfoOutput {
	channelsInfo := make([]*model.ChannelInfoOutput, 0)
	for _, channel := range raw {
		tmp := MappingChannelInfoFromRaw(channel)
		channelsInfo = append(channelsInfo, tmp)
	}
	return channelsInfo
}

func MappingChannelInfoFromRaw(raw *RawChannelRating) *model.ChannelInfoOutput {
	return &model.ChannelInfoOutput{
		Age:            int(raw.Age),
		ChannelID:      raw.ChannelID,
		NodeID:         raw.NodeID,
		Alias:          raw.Alias,
		Direction:      raw.Direction,
		Capacity:       raw.Capacity,
		Fee:            raw.Fee,
		Limits:         raw.Limits,
		UpTime:         MappingUpTimeRatingFromRaw(raw.UpTimeRating),
		ForwardsRating: MappingForwardsRatingFromRaw(raw.ForwardsRating),
	}
}
