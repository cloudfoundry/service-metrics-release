package metrics

type GaugeMetric struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type CounterMetric struct {
	Name  string `json:"name"`
	Delta uint64 `json:"delta"`
}
