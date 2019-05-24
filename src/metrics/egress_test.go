package metrics_test

import (
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/service-metrics/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Egress", func() {
	var (
		m  []metrics.GaugeMetric
		l  *spyLogger
		in *spyIngressClient
		c  *metrics.EgressClient
	)

	BeforeEach(func() {
		m = []metrics.GaugeMetric{
			{
				Key:   "metric-1",
				Value: 0.1,
				Unit:  "s",
			},
			{
				Key:   "metric-2",
				Value: 1.3,
				Unit:  "s",
			},
		}

		l = newSpyLogger()
		in = newSpyIngressClient()
		c = metrics.NewEgressClient(in, "source-1")
	})

	Context("Emit", func() {
		It("passes logs to IngressClient", func() {
			c.SetInstanceID(3)
			c.EmitGauges(m, l)

			Expect(in.emitGaugeCalled).To(BeTrue())

			env := in.gaugeEnvs[0]
			Expect(env.GetGauge().Metrics).To(HaveKeyWithValue("metric-2",
				&loggregator_v2.GaugeValue{Value: 1.3, Unit: "s"}),
			)
			Expect(env.GetGauge().Metrics).To(HaveKeyWithValue("metric-1",
				&loggregator_v2.GaugeValue{Value: 0.1, Unit: "s"}),
			)
			Expect(env.SourceId).To(Equal("source-1"))
			Expect(env.InstanceId).To(Equal("3"))
			Expect(l.infoKey).To(Equal("sending-metrics"))
		})

		It("logs an error when source ID is not specified", func() {
			c = metrics.NewEgressClient(in, "")
			c.EmitGauges(m, l)

			Expect(l.errAction).To(Equal("sending metrics failed"))
			Expect(l.errData).To(ConsistOf(
				lager.Data{
					"Emit": "failed",
				},
			))
			Expect(l.err).To(MatchError("You must set a source ID"))
		})
	})
})

type spyIngressClient struct {
	emitGaugeCalled   bool
	emitCounterCalled bool
	gaugeEnvs         []*loggregator_v2.Envelope
	counterEnvs       []*loggregator_v2.Envelope
}

func newSpyIngressClient() *spyIngressClient {
	return &spyIngressClient{}
}

func (s *spyIngressClient) EmitGauge(opts ...loggregator.EmitGaugeOption) {
	s.emitGaugeCalled = true
	s.gaugeEnvs = append(s.gaugeEnvs, gaugeEnv(opts))
}

func (s *spyIngressClient) EmitCounter(name string, opts ...loggregator.EmitCounterOption) {
	s.emitCounterCalled = true
	s.counterEnvs = append(s.counterEnvs, counterEnv(name, opts))
}

func gaugeEnv(opts []loggregator.EmitGaugeOption) *loggregator_v2.Envelope {
	env := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: make(map[string]*loggregator_v2.GaugeValue),
			},
		},
		Tags: make(map[string]string),
	}
	for _, o := range opts {
		o(env)
	}

	return env
}

func counterEnv(name string, opts []loggregator.EmitCounterOption) *loggregator_v2.Envelope {
	env := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name: name,
			},
		},
		Tags: make(map[string]string),
	}
	for _, o := range opts {
		o(env)
	}

	return env
}
