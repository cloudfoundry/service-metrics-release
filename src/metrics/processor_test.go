package metrics_test

import (
	"fmt"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/service-metrics/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Processor", func() {
	It("sends gauge metrics to the egress client", func() {
		spyIngressClient := newSpyIngressClient()
		spyExecutor := newSpyExecutor([]byte(`[
			{"key": "my-key", "value": 21.4, "unit": "things"},
			{"key": "my-other-key", "value": 39.9, "unit": "other-things"}
		]`), nil)

		p := metrics.NewProcessor(
			&spyLogger{},
			metrics.NewEgressClient(spyIngressClient, "source-id"),
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))
		Expect(spyIngressClient.gaugeEnvs).To(HaveLen(1))

		ms := spyIngressClient.gaugeEnvs[0].GetGauge().GetMetrics()
		Expect(ms["my-key"].GetValue()).To(Equal(21.4))
		Expect(ms["my-key"].GetUnit()).To(Equal("things"))
	})

	It("doesn't emit gauges when the output isn't a gauge", func() {
		invalidMetrics := []string{
			`{"not-key": "my-key", "value": 21.4, "unit":"things"}`,     // Invalid key name
			`{"key": "my-key", "not-value": 21.4, "unit":"things"}`,     // Invalid value name
			`{"key": "my-key", "value": 21.4, "not-unit":"things"}`,     // Invalid unit name
			`{"key": 0, "value": 21.4, "unit":"things"}`,                // Invalid key value
			`{"key": "my-key", "value": "not number", "unit":"things"}`, // Invalid value value
			`{"key": "my-key", "value": 21.4, "unit":0}`,                // Invalid unit value
		}
		out := fmt.Sprintf("[%s]", strings.Join(invalidMetrics, ","))

		spyIngressClient := newSpyIngressClient()
		spyExecutor := newSpyExecutor([]byte(out), nil)

		p := metrics.NewProcessor(
			&spyLogger{},
			metrics.NewEgressClient(spyIngressClient, "source-id"),
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyIngressClient.emitGaugeCalled).To(BeFalse())
	})

	It("sends counter metrics to the egress client", func() {
		spyIngressClient := newSpyIngressClient()
		spyExecutor := newSpyExecutor([]byte(`[
			{"name": "my-name", "delta": 1},
			{"name": "my-other-name", "delta": 14}
		]`), nil)

		p := metrics.NewProcessor(
			&spyLogger{},
			metrics.NewEgressClient(spyIngressClient, "source-id"),
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyIngressClient.counterEnvs).To(HaveLen(2))

		env0 := spyIngressClient.counterEnvs[0]
		Expect(env0.GetCounter().GetName()).To(Equal("my-name"))
		Expect(env0.GetCounter().GetDelta()).To(Equal(uint64(1)))

		env1 := spyIngressClient.counterEnvs[1]
		Expect(env1.GetCounter().GetName()).To(Equal("my-other-name"))
		Expect(env1.GetCounter().GetDelta()).To(Equal(uint64(14)))
	})

	It("doesn't emit counters when the output isn't a counter", func() {
		invalidMetrics := []string{
			`{"not-name": "my-name", "delta": 1}`,    // Inalid name name
			`{"name": "my-name", "not-delta": 1}`,    // Invalid delta name
			`{"name": 0, "delta": 1}`,                // Invalid name value
			`{"name": "my-name", "delta": "number"}`, // Invalid delta value
		}
		out := fmt.Sprintf("[%s]", strings.Join(invalidMetrics, ","))

		spyIngressClient := newSpyIngressClient()
		spyExecutor := newSpyExecutor([]byte(out), nil)

		p := metrics.NewProcessor(
			&spyLogger{},
			metrics.NewEgressClient(spyIngressClient, "source-id"),
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyIngressClient.counterEnvs).To(HaveLen(0))
	})

	It("ignores counters with negative values", func() {
		spyIngressClient := newSpyIngressClient()
		spyExecutor := newSpyExecutor([]byte(`[
			{"name": "my-name", "delta": -1}
		]`), nil)

		p := metrics.NewProcessor(
			&spyLogger{},
			metrics.NewEgressClient(spyIngressClient, "source-id"),
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyIngressClient.emitCounterCalled).To(BeFalse())
	})
})

type spyExecutor struct {
	cmd *exec.Cmd
	out []byte
	err error
}

func newSpyExecutor(o []byte, e error) *spyExecutor {
	return &spyExecutor{
		out: o,
		err: e,
	}
}

func (e *spyExecutor) Run(c *exec.Cmd) ([]byte, error) {
	e.cmd = c

	return e.out, e.err
}
