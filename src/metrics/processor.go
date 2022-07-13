package metrics

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"

	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/lager"
)

var (
	invalidNameRegex = regexp.MustCompile(`[^a-zA-Z0-9_:]`)
)

type Executor interface {
	Run(*exec.Cmd) ([]byte, error)
}

type Logger interface {
	Info(string, ...lager.Data)
	Error(action string, err error, data ...lager.Data)
}

type Processor struct {
	logger   Logger
	executor Executor
	metrics  metricsRegistry
}

type metricsRegistry interface {
	NewCounter(name, helpText string, opts ...metrics.MetricOption) metrics.Counter
	NewGauge(name, helpText string, opts ...metrics.MetricOption) metrics.Gauge
}

func NewProcessor(l Logger, m metricsRegistry, e Executor) Processor {
	return Processor{
		logger:   l,
		metrics:  m,
		executor: e,
	}
}

func (p *Processor) Process(cmdPath string, args ...string) {
	out, err := p.executor.Run(exec.Command(cmdPath, args...))
	if err != nil {
		return
	}

	var parsedMetrics []map[string]interface{}
	err = json.NewDecoder(bytes.NewReader(out)).Decode(&parsedMetrics)
	if err != nil {
		p.logger.Error("parsing-metrics-output", err, lager.Data{
			"event":  "failed",
			"output": string(out),
		})
		os.Exit(1)
	}

	for _, metric := range parsedMetrics {
		if isGauge(metric) {
			p.recordGauge(metric)
			continue
		}

		if isCounter(metric) {
			p.recordCounter(metric)
		}
	}
}

func (p *Processor) recordGauge(metric map[string]interface{}) {
	sanitizeName, modified := sanitizeName(metric["key"].(string))
	if modified {
		p.metrics.NewCounter("modified_metric_name", "").Add(1.0)
	}

	p.metrics.NewGauge(
		sanitizeName,
		"",
		metrics.WithMetricLabels(
			map[string]string{"unit": metric["unit"].(string)},
		),
	).Set(metric["value"].(float64))
}

func (p *Processor) recordCounter(metric map[string]interface{}) {
	sanitizeName, modified := sanitizeName(metric["name"].(string))
	if modified {
		p.metrics.NewCounter("modified_metric_name", "").Add(1.0)
	}

	p.metrics.NewCounter(
		sanitizeName,
		"",
	).Add(metric["delta"].(float64))
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

func sanitizeName(name string) (string, bool) {
	sanitized := invalidNameRegex.ReplaceAllString(name, "_")
	return sanitized, sanitized != name
}
