package metric_one

import (
	"encoding/json"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/config"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
	"strings"
	"time"
)

// CalculateMetricOneOutputSync Method to calculate the metric one output and store the result
// on the server without channels but in a synchronous way.
func CalculateMetricOneOutputSync(storage db.MetricsDatabase, metricModel *model.MetricOne) error {
	metricKey := strings.Join([]string{metricModel.NodeID, config.RawMetricOnePrefix}, "/")
	log.GetInstance().Infof("Raw metric for node %s key output is: %s", metricModel.NodeID, metricKey)
	rawMetric, err := storage.GetRawValue(metricKey)
	var rawMetricModel *RawMetricOneOutput
	if err != nil {
		// If the metrics it is not found, create a new one
		// and continue to fill it.
		rawMetricModel = NewRawMetricOneOutput(time.Now().Unix())
	} else {
		if err := json.Unmarshal(rawMetric, &rawMetricModel); err != nil {
			return err
		}
	}

	// FIXME: split also the metrics inside different metrics, calculated by
	// timestamp!
	//
	// Make intersection between channels info
	// this give the possibility to remove from the raw metrics
	// channels that are no longer available
	if err := intersectionChannelsInfo(metricModel, rawMetricModel); err != nil {
		log.GetInstance().Errorf("Error: %s", err)
		return nil
	}

	CalculateUptimeMetricOneSync(storage, rawMetricModel, metricModel)
	CalculateForwardsRatingMetricOneSync(storage, rawMetricModel.ForwardsRating, metricModel)
	itemKey, _ := storage.ItemID(metricModel)
	CalculateRationForChannelsSync(storage, itemKey, rawMetricModel.ChannelsRating, metricModel.ChannelsInfo)

	rawMetricModel.LastUpdate = time.Now().Unix()
	// As result, we store the value on the db
	metricModelBytes, err := json.Marshal(rawMetricModel)
	if err != nil {
		return err
	}

	if err := storage.PutRawValue(metricKey, metricModelBytes); err != nil {
		return err
	}
	resKey := strings.Join([]string{metricModel.NodeID, config.MetricOneOutputSuffix}, "/")

	result := CalculateMetricOneOutputFromRaw(rawMetricModel)

	resultByte, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return storage.PutRawValue(resKey, resultByte)
}
