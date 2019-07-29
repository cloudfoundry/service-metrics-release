package main

import (
	egress "code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/service-metrics/metrics"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/go-envstruct"
)

type config struct {
	Origin          string        `env:"ORIGIN"`
	SourceID        string        `env:"SOURCE_ID"`
	MetricsInterval time.Duration `env:"METRICS_INTERVAL"`
	MetricsCmd      string        `env:"METRICS_CMD"`
	MetricsCmdArgs  multiFlag     `env:"METRICS_CMD_ARG"`
	Debug           bool          `env:"DEBUG"`
	Port            int           `env:"PORT"`
}

var cfg config

func main() {
	parseConfig()

	stdoutLogLevel := lager.INFO
	if cfg.Debug {
		stdoutLogLevel = lager.DEBUG
	}

	logger := lager.NewLogger("service-metrics")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, stdoutLogLevel))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	m := egress.NewRegistry(log.New(os.Stdout, "", 0),
		egress.WithDefaultTags(
			map[string]string{
				"origin":    cfg.Origin,
				"source_id": cfg.SourceID,
			},
		),
		egress.WithServer(cfg.Port),
	)

	processor := metrics.NewProcessor(
		logger,
		m,
		NewCommandLineExecutor(logger),
	)

	processor.Process(cfg.MetricsCmd, cfg.MetricsCmdArgs...)
	for {
		select {
		case <-time.After(cfg.MetricsInterval):
			processor.Process(cfg.MetricsCmd, cfg.MetricsCmdArgs...)
		}
	}
}

func parseConfig() {
	env := &config{
		MetricsInterval: time.Minute,
	}
	envstruct.Load(env)

	flag.StringVar(&cfg.Origin, "origin", "", "Required. Source name for metrics emitted by this process, e.g. service-name")
	flag.StringVar(&cfg.MetricsCmd, "metrics-cmd", "", "Required. Path to metrics command")
	flag.StringVar(&cfg.SourceID, "source-id", "", "Source ID to be applied to all envelopes.")
	flag.Var(&cfg.MetricsCmdArgs, "metrics-cmd-arg", "Argument to pass on to metrics-cmd (multi-valued)")
	flag.DurationVar(&cfg.MetricsInterval, "metrics-interval", 0, "Interval to run metrics-cmd")
	flag.BoolVar(&cfg.Debug, "debug", false, "Output debug logging")
	flag.Parse()

	if cfg.Origin == "" {
		cfg.Origin = env.Origin
	}

	if cfg.MetricsCmd == "" {
		cfg.MetricsCmd = env.MetricsCmd
	}

	if cfg.SourceID == "" {
		cfg.SourceID = env.SourceID
	}

	if len(cfg.MetricsCmdArgs) == 0 {
		cfg.MetricsCmdArgs = env.MetricsCmdArgs
	}

	if cfg.MetricsInterval == 0 {
		cfg.MetricsInterval = env.MetricsInterval
	}

	if cfg.Debug {
		cfg.Debug = env.Debug
	}

	if cfg.SourceID == "" {
		cfg.SourceID = cfg.Origin
	}

	cfg.Port = env.Port

	assertFlag("origin", cfg.Origin)
	assertFlag("metrics-cmd", cfg.MetricsCmd)
}

type multiFlag []string

// multiFlag implements flag.Value
func (m *multiFlag) String() string {
	return fmt.Sprint(cfg.MetricsCmdArgs)
}

// multiFlag implements flag.Value
func (m *multiFlag) Set(value string) error {
	if cfg.MetricsCmdArgs == nil {
		cfg.MetricsCmdArgs = multiFlag{}
	}

	cfg.MetricsCmdArgs = append(cfg.MetricsCmdArgs, value)

	return nil
}

// multiFlag implements envstruct.Unmarshaller
func (m *multiFlag) UnmarshalEnv(v string) error {
	cfg.MetricsCmdArgs = multiFlag{v}
	return nil
}

func assertFlag(name, value string) {
	if value == "" {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "\nMust provide --%s", name)
		os.Exit(1)
	}
}

type logWrapper struct {
	lager.Logger
}

func (l *logWrapper) Printf(f string, a ...interface{}) {
	l.Info(fmt.Sprintf(f, a...))
}

func (l *logWrapper) Panicf(f string, a ...interface{}) {
	l.Fatal(fmt.Sprintf(f, a...), nil)
}
