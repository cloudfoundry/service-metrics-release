package metrics

import (
	"errors"
	"strconv"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/lager"
)

type Logger interface {
	Info(string, ...lager.Data)
	Error(action string, err error, data ...lager.Data)
}

type IngressClient interface {
	EmitGauge(opts ...loggregator.EmitGaugeOption)
	EmitCounter(name string, opts ...loggregator.EmitCounterOption)
}

type EgressClient struct {
	emitter    IngressClient
	sourceID   string
	instanceID string
}

func NewEgressClient(inClient IngressClient, sourceID string) *EgressClient {
	return &EgressClient{
		emitter:  inClient,
		sourceID: sourceID,
	}
}

func (c *EgressClient) EmitCounters(metrics []CounterMetric, logger Logger) {
	if len(metrics) < 1 {
		logger.Info("sending-metrics", lager.Data{
			"details": "no metrics to send",
		})
		return
	}

	if c.sourceID == "" {
		e := errors.New("You must set a source ID")
		logger.Error("sending metrics failed", e, lager.Data{
			"Emit": "failed",
		})
	}

	logger.Info("sending-metrics", lager.Data{
		"details": "emitting counters to logging platform",
		"count":   len(metrics),
	})

	for _, m := range metrics {
		opts := []loggregator.EmitCounterOption{
			loggregator.WithCounterSourceInfo(c.sourceID, c.instanceID),
			loggregator.WithDelta(m.Delta),
		}
		c.emitter.EmitCounter(m.Name, opts...)
	}

}

func (c *EgressClient) EmitGauges(metrics []GaugeMetric, logger Logger) {
	if len(metrics) < 1 {
		logger.Info("sending-metrics", lager.Data{
			"details": "no metrics to send",
		})
		return
	}

	if c.sourceID == "" {
		e := errors.New("You must set a source ID")
		logger.Error("sending metrics failed", e, lager.Data{
			"Emit": "failed",
		})
	}

	logger.Info("sending-metrics", lager.Data{
		"details": "emitting gauges to logging platform",
		"count":   len(metrics),
	})

	opts := []loggregator.EmitGaugeOption{
		loggregator.WithGaugeSourceInfo(c.sourceID, c.instanceID),
	}

	for _, m := range metrics {
		opts = append(opts, loggregator.WithGaugeValue(m.Key, m.Value, m.Unit))
	}

	c.emitter.EmitGauge(opts...)
}

func (c *EgressClient) SetInstanceID(instanceID int) {
	c.instanceID = strconv.Itoa(instanceID)
}
