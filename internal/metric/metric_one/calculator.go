package metric_one

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/pkg/utils"
)

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
	// FIXME: move this to local failure
	tot := raw.Success + raw.Failure + raw.InternalFailure
	return &model.ForwardsRating{
		Success:         utils.Percentage(raw.Success, tot),
		Failure:         utils.Percentage(raw.Failure, tot),
		InternalFailure: utils.Percentage(raw.InternalFailure, tot),
		LocalFailure:    utils.Percentage(raw.LocalFailure, tot),
	}
}

func MappingUpTimeRatingFromRaw(raw *RawPercentageData) *model.UpTimeOutput {
	return &model.UpTimeOutput{
		OneDay:     utils.Percentage(raw.TodaySuccess, raw.TodayTotal),
		TenDays:    utils.Percentage(raw.TenDaysSuccess, raw.TenDaysTotal),
		ThirtyDays: utils.Percentage(raw.ThirtyDaysSuccess, raw.ThirtyDaysTotal),
		SixMonths:  utils.Percentage(raw.SixMonthsSuccess, raw.SixMonthsTotal),
		Full:       utils.Percentage(raw.FullSuccess, raw.FullTotal),
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
