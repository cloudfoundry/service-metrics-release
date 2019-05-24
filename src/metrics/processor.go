package metrics

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

type Executor interface {
	Run(*exec.Cmd) ([]byte, error)
}

type Processor struct {
	client   *EgressClient
	logger   Logger
	executor Executor
}

func NewProcessor(l Logger, c *EgressClient, e Executor) Processor {
	return Processor{
		logger:   l,
		client:   c,
		executor: e,
	}
}

func (p *Processor) Process(cmdPath string, args ...string) {
	out, err := p.executor.Run(exec.Command(cmdPath, args...))
	if err != nil {
		return
	}

	var (
		parsedGauges   []GaugeMetric
		parsedCounters []CounterMetric
		parsedMetrics  []map[string]interface{}
	)

	err = json.NewDecoder(bytes.NewReader(out)).Decode(&parsedMetrics)
	if err != nil {
		p.logger.Error("parsing-metrics-output", err, lager.Data{
			"event":  "failed",
			"output": string(out),
		})
		os.Exit(1)
	}

	for _, i := range parsedMetrics {
		if isGauge(i) {
			parsedGauges = append(parsedGauges, GaugeMetric{
				Key:   i["key"].(string),
				Value: i["value"].(float64),
				Unit:  i["unit"].(string),
			})
			continue
		}

		if isCounter(i) {
			parsedCounters = append(parsedCounters, CounterMetric{
				Name:  i["name"].(string),
				Delta: uint64(i["delta"].(float64)),
			})
		}
	}

	p.client.EmitGauges(parsedGauges, p.logger)
	p.client.EmitCounters(parsedCounters, p.logger)
}

func isGauge(m map[string]interface{}) bool {
	if !hasStringKey(m, "key") {
		return false
	}

	if !hasFloat64Key(m, "value") {
		return false
	}

	if !hasStringKey(m, "unit") {
		return false
	}

	return true
}

func isCounter(m map[string]interface{}) bool {
	if !hasStringKey(m, "name") {
		return false
	}

	if !hasFloat64Key(m, "delta") {
		return false
	}

	if m["delta"].(float64) < 0 {
		return false
	}

	return true
}

func hasStringKey(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}

	if _, ok = v.(string); !ok {
		return false
	}

	return true
}

func hasFloat64Key(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}

	if _, ok = v.(float64); !ok {
		return false
	}

	return true
}
