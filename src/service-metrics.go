package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/service-metrics/metrics"
	envstruct "github.com/cloudfoundry/go-envstruct"
)

type config struct {
	Origin          string        `env:"ORIGIN"`
	SourceID        string        `env:"SOURCE_ID"`
	AgentAddr       string        `env:"AGENT_ADDR"`
	MetricsInterval time.Duration `env:"METRICS_INTERVAL"`
	MetricsCmd      string        `env:"METRICS_CMD"`
	MetricsCmdArgs  multiFlag     `env:"METRICS_CMD_ARG"`
	Debug           bool          `env:"DEBUG"`
	CaPath          string        `env:"CA_PATH"`
	CertPath        string        `env:"CERT_PATH"`
	KeyPath         string        `env:"KEY_PATH"`
}

var cfg config

func main() {
	env := &config{
		MetricsInterval: time.Minute,
	}

	envstruct.Load(env)

	flag.StringVar(&cfg.Origin, "origin", "", "Required. Source name for metrics emitted by this process, e.g. service-name")
	flag.StringVar(&cfg.AgentAddr, "agent-addr", "", "Required. Loggregator agent address, e.g. localhost:2346")
	flag.StringVar(&cfg.MetricsCmd, "metrics-cmd", "", "Required. Path to metrics command")
	flag.StringVar(&cfg.CaPath, "ca", "", "Required. Path to CA certificate")
	flag.StringVar(&cfg.CertPath, "cert", "", "Required. Path to client TLS certificate")
	flag.StringVar(&cfg.KeyPath, "key", "", "Required. Path to client TLS private key")
	flag.StringVar(&cfg.SourceID, "source-id", "", "Source ID to be applied to all envelopes.")
	flag.Var(&cfg.MetricsCmdArgs, "metrics-cmd-arg", "Argument to pass on to metrics-cmd (multi-valued)")
	flag.DurationVar(&cfg.MetricsInterval, "metrics-interval", 0, "Interval to run metrics-cmd")
	flag.BoolVar(&cfg.Debug, "debug", false, "Output debug logging")

	flag.Parse()

	if cfg.Origin == "" {
		cfg.Origin = env.Origin
	}

	if cfg.AgentAddr == "" {
		cfg.AgentAddr = env.AgentAddr
	}

	if cfg.MetricsCmd == "" {
		cfg.MetricsCmd = env.MetricsCmd
	}

	if cfg.CaPath == "" {
		cfg.CaPath = env.CaPath
	}

	if cfg.CertPath == "" {
		cfg.CertPath = env.CertPath
	}

	if cfg.KeyPath == "" {
		cfg.KeyPath = env.KeyPath
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

	assertFlag("origin", cfg.Origin)
	assertFlag("agent-addr", cfg.AgentAddr)
	assertFlag("metrics-cmd", cfg.MetricsCmd)
	assertFlag("ca", cfg.CaPath)
	assertFlag("cert", cfg.CertPath)
	assertFlag("key", cfg.KeyPath)

	if cfg.SourceID == "" {
		cfg.SourceID = cfg.Origin
	}

	stdoutLogLevel := lager.INFO
	if cfg.Debug {
		stdoutLogLevel = lager.DEBUG
	}

	logger := lager.NewLogger("service-metrics")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, stdoutLogLevel))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	tlsConfig, err := loggregator.NewIngressTLSConfig(cfg.CaPath, cfg.CertPath, cfg.KeyPath)
	if err != nil {
		logger.Error("Failed to load TLS config", err)
		os.Exit(1)
	}

	loggregatorClient, err := loggregator.NewIngressClient(tlsConfig,
		loggregator.WithAddr(cfg.AgentAddr),
		loggregator.WithLogger(&logWrapper{logger}),
		loggregator.WithTag("origin", cfg.Origin),
	)
	if err != nil {
		logger.Error("Failed to initialize loggregator client", err)
		os.Exit(1)
	}

	egressClient := metrics.NewEgressClient(loggregatorClient, cfg.SourceID)
	processor := metrics.NewProcessor(
		logger,
		egressClient,
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
