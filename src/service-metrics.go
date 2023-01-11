package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	egress "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/service-metrics-release/metrics"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/lager/v3"
)

type config struct {
	Origin          string        `env:"ORIGIN, report"`
	MetricsInterval time.Duration `env:"METRICS_INTERVAL, report"`
	MetricsCmd      string        `env:"METRICS_CMD, report"`
	MetricsCmdArgs  multiFlag     `env:"METRICS_CMD_ARG"`
	Debug           bool          `env:"DEBUG, report"`
	Port            int           `env:"PORT, report"`
	CAFile          string        `env:"CA_FILE_PATH, report"`
	CertFile        string        `env:"CERT_FILE_PATH, report"`
	KeyFile         string        `env:"KEY_FILE_PATH, report"`
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
		egress.WithTLSServer(
			cfg.Port,
			cfg.CertFile,
			cfg.KeyFile,
			cfg.CAFile,
		),
	)

	processor := metrics.NewProcessor(
		logger,
		m,
		NewCommandLineExecutor(logger),
	)

	processor.Process(cfg.MetricsCmd, cfg.MetricsCmdArgs...)
	for {
		<-time.After(cfg.MetricsInterval)
		processor.Process(cfg.MetricsCmd, cfg.MetricsCmdArgs...)
	}
}

func parseConfig() {
	cfg = config{
		MetricsInterval: time.Minute,
	}
	err := envstruct.Load(&cfg)
	if err != nil {
		log.Panicf("error loading envstruct: %s", err)
	}

	cmdArgsFromEnv := cfg.MetricsCmdArgs
	flag.StringVar(&cfg.Origin, "origin", cfg.Origin, "Required. Source name for metrics emitted by this process, e.g. service-name")
	flag.StringVar(&cfg.MetricsCmd, "metrics-cmd", cfg.MetricsCmd, "Required. Path to metrics command")
	flag.Var(&cfg.MetricsCmdArgs, "metrics-cmd-arg", "Argument to pass on to metrics-cmd (multi-valued)")
	flag.DurationVar(&cfg.MetricsInterval, "metrics-interval", cfg.MetricsInterval, "Interval to run metrics-cmd")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "Output debug logging")
	flag.Parse()

	if len(cfg.MetricsCmdArgs) == 0 {
		cfg.MetricsCmdArgs = cmdArgsFromEnv
	}

	assertFlag("origin", cfg.Origin)
	assertFlag("metrics-cmd", cfg.MetricsCmd)

	err = envstruct.WriteReport(&cfg)
	if err != nil {
		log.Panicf("error writing report: %s", err)
	}
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
