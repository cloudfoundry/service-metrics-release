package metrics_test

import (
	"code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/service-metrics/metrics"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Processor", func() {
	It("sends gauge metrics to the egress client", func() {
		spyExecutor := newSpyExecutor([]byte(`[
			{"key": "my-key", "value": 21.4, "unit": "things"},
			{"key": "my-other-key", "value": 39.9, "unit": "other-things"}
		]`), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))

		Eventually(func() float64 {
			return m.GetMetricValue("my_key", map[string]string{"unit": "things"})
		}).Should(Equal(21.4))
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

		spyExecutor := newSpyExecutor([]byte(out), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")
		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))

		Expect(m.Metrics).To(HaveLen(0))
	})

	It("sends counter metrics to the egress client", func() {
		spyExecutor := newSpyExecutor([]byte(`[
			{"name": "my-name", "delta": 1},
			{"name": "my-other-name", "delta": 14}
		]`), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(m.GetMetricValue("my_name", nil)).To(Equal(1.0))
		Expect(m.GetMetricValue("my_other_name", nil)).To(Equal(14.0))
	})

	It("doesn't emit counters when the output isn't a counter", func() {
		invalidMetrics := []string{
			`{"not-name": "my-name", "delta": 1}`,    // Invalid name name
			`{"name": "my-name", "not-delta": 1}`,    // Invalid delta name
			`{"name": 0, "delta": 1}`,                // Invalid name value
			`{"name": "my-name", "delta": "number"}`, // Invalid delta value
		}
		out := fmt.Sprintf("[%s]", strings.Join(invalidMetrics, ","))

		spyExecutor := newSpyExecutor([]byte(out), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")
		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))

		Expect(m.Metrics).To(HaveLen(0))
	})

	It("ignores counters with negative values", func() {
		spyExecutor := newSpyExecutor([]byte(`[
			{"name": "my-name", "delta": -1}
		]`), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")
		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))

		Expect(m.Metrics).To(HaveLen(0))
	})

	It("converts names with invalid characters", func() {
		spyExecutor := newSpyExecutor([]byte(`[
			{"key": "gauge.wrong.name", "value": 21.4, "unit": "things"},
			{"name": "counter/also-wrong", "delta": 1}
		]`), nil)

		m := testhelpers.NewMetricsRegistry()
		p := metrics.NewProcessor(
			&spyLogger{},
			m,
			spyExecutor,
		)

		p.Process("/bin/echo", "my", "command")

		Expect(spyExecutor.cmd.Args).To(Equal([]string{"/bin/echo", "my", "command"}))

		Eventually(func() float64 {
			return m.GetMetricValue("gauge_wrong_name", map[string]string{"unit": "things"})
		}).Should(Equal(21.4))

		Eventually(func() float64 {
			return m.GetMetricValue("counter_also_wrong", nil)
		}).Should(Equal(1.0))
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
