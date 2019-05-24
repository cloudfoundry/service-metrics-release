package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/st3v/glager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var _ = Describe("service-metrics", func() {

	var (
		origin          string
		sourceID        string
		agentAddr       string
		metricsCmd      string
		metricsCmdArgs  []string
		metricsInterval string
		debugLog        bool
		session         *gexec.Session
		caPath          string
		certPath        string
		keyPath         string
		stubAgent       *stubAgent
		envVars         []string
	)

	var metricsJson = `
		[
			{
				"key": "loadMetric",
				"value": 4,
				"unit": "Load"
			},
			{
				"key": "temperatureMetric",
				"value": 99,
				"unit": "Temperature"
			}
		]
	`

	BeforeEach(func() {
		stubAgent = newStubAgent()
		origin = "p-service-origin"
		sourceID = "my-source-id"
		debugLog = false
		agentAddr = stubAgent.address
		metricsCmd = "/bin/echo"
		metricsCmdArgs = []string{"-n", metricsJson}
		metricsInterval = "10ms"
		caPath = Cert("loggregator-ca.crt")
		certPath = Cert("service-metrics.crt")
		keyPath = Cert("service-metrics.key")
		envVars = nil
	})

	JustBeforeEach(func() {
		session = runCmd(
			origin,
			sourceID,
			debugLog,
			agentAddr,
			metricsInterval,
			metricsCmd,
			caPath,
			certPath,
			keyPath,
			metricsCmdArgs,
			envVars,
		)
	})

	AfterEach(func() {
		Eventually(session.Interrupt()).Should(gexec.Exit())
	})

	Context("when loggregator-agent is running", func() {
		It("repeatedly emits metrics", func() {
			Eventually(func() int {
				return len(stubAgent.GetEnvelopes())
			}).Should(BeNumerically(">", 5))
			env := stubAgent.GetEnvelopes()[0]

			Expect(env.GetGauge().GetMetrics()).To(HaveKeyWithValue("loadMetric",
				&loggregator_v2.GaugeValue{Value: 4.0, Unit: "Load"},
			))
			Expect(env.GetGauge().GetMetrics()).To(HaveKeyWithValue("temperatureMetric",
				&loggregator_v2.GaugeValue{Value: 99, Unit: "Temperature"},
			))
			Expect(env.Tags["origin"]).To(Equal("p-service-origin"))
			Expect(env.SourceId).To(Equal("my-source-id"))
		})

		Context("when no flags are provided", func() {
			BeforeEach(func() {
				origin = ""
				sourceID = ""
				debugLog = false
				metricsCmd = ""
				metricsCmdArgs = nil
				metricsInterval = ""
				caPath = ""
				certPath = ""
				keyPath = ""

				envVars = []string{
					"ORIGIN=p-service-origin",
					"SOURCE_ID=my-source-id",
					fmt.Sprint("AGENT_ADDR=", stubAgent.address),
					"METRICS_INTERVAL=10ms",
					"METRICS_CMD=/bin/echo",
					fmt.Sprint("METRICS_CMD_ARG=", metricsJson),
					"DEBUG=false",
					fmt.Sprint("CA_PATH=", Cert("loggregator-ca.crt")),
					fmt.Sprint("CERT_PATH=", Cert("service-metrics.crt")),
					fmt.Sprint("KEY_PATH=", Cert("service-metrics.key")),
				}
			})

			It("repeatedly emits metrics", func() {
				Eventually(func() int {
					return len(stubAgent.GetEnvelopes())
				}).Should(BeNumerically(">", 5))
				env := stubAgent.GetEnvelopes()[0]

				Expect(env.GetGauge().GetMetrics()).To(HaveKeyWithValue("loadMetric",
					&loggregator_v2.GaugeValue{Value: 4.0, Unit: "Load"},
				))
				Expect(env.GetGauge().GetMetrics()).To(HaveKeyWithValue("temperatureMetric",
					&loggregator_v2.GaugeValue{Value: 99, Unit: "Temperature"},
				))
				Expect(env.Tags["origin"]).To(Equal("p-service-origin"))
				Expect(env.SourceId).To(Equal("my-source-id"))
			})
		})
	})

	Context("when no source ID is given", func() {
		BeforeEach(func() {
			sourceID = ""
		})

		It("uses the origin as the source ID", func() {
			Eventually(func() int {
				return len(stubAgent.GetEnvelopes())
			}).Should(BeNumerically(">", 5))
			env := stubAgent.GetEnvelopes()[0]

			Expect(env.SourceId).To(Equal("p-service-origin"))
		})
	})

	It("never exits", func() {
		Consistently(session.ExitCode).Should(Equal(-1))
	})

	It("logs the call to metrics command to stdout", func() {
		Eventually(func() *gbytes.Buffer {
			return session.Out
		}).Should(ContainSequence(
			Info(
				Message("service-metrics.executing-metrics-cmd"),
				Data("event", "starting"),
			),
			Info(
				Message("service-metrics.executing-metrics-cmd"),
				Data("event", "done"),
			),
		))
	})

	Context("when the metrics command exits with 1", func() {
		BeforeEach(func() {
			metricsCmd = "/bin/bash"
			metricsCmdArgs = []string{"-c", "echo -n failed to obtain metrics; exit 1"}
		})

		It("exits with an exit code of zero", func() {
			Eventually(session.ExitCode).Should(Equal(0))
		})

		It("logs the error to stdout", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Out
			}).Should(ContainSequence(
				Error(
					AnyErr,
					Message("service-metrics.executing-metrics-cmd"),
					Data("event", "failed"),
					Data("output", "failed to obtain metrics"),
				),
			))
		})
	})

	Context("when the metrics command exits with 10", func() {
		BeforeEach(func() {
			metricsCmd = "/bin/bash"
			metricsCmdArgs = []string{"-c", "echo -n failed to obtain metrics; exit 10"}
		})

		It("never exits", func() {
			Consistently(session.ExitCode).Should(Equal(-1))
		})

		It("logs not ready to emit metrics and the metrics command output to stdout", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Out
			}).Should(ContainSequence(
				Info(
					Message("service-metrics.executing-metrics-cmd"),
					Data("event", "not yet ready to emit metrics"),
					Data("output", "failed to obtain metrics"),
				),
			))
		})
	})

	Context("when the metrics command returns invalid JSON", func() {
		BeforeEach(func() {
			metricsCmd = "/bin/echo"
			metricsCmdArgs = []string{"-n", "invalid"}
		})

		It("exits with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("logs a fatal error to stdout", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Out
			}).Should(ContainSequence(
				Error(
					errors.New("invalid character 'i' looking for beginning of value"),
					Message("service-metrics.parsing-metrics-output"),
					Data("event", "failed"),
					Data("output", "invalid"),
				),
			))
		})
	})

	Context("when the metrics command does not exist", func() {
		BeforeEach(func() {
			metricsCmd = "/your/system/wont/have/this/yet"
			metricsCmdArgs = []string{}
		})

		It("exits with an exit code of 1", func() {
			Eventually(session.ExitCode).Should(Equal(1))
		})

		It("logs the error to stdout", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Out
			}).Should(ContainSequence(
				Error(
					AnyErr,
					Message("service-metrics.executing-metrics-cmd"),
					Data("event", "failed"),
					Data("output", "no metrics command has been configured, cannot collect metrics"),
				),
			))
		})
	})

	Context("when the --origin param is not provided", func() {
		BeforeEach(func() {
			origin = ""
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --origin"))
		})
	})

	Context("when the --metrics-cmd param is not provided", func() {
		BeforeEach(func() {
			metricsCmd = ""
			metricsCmdArgs = []string{}
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --metrics-cmd"))
		})
	})

	Context("when the --agent-addr param is not provided", func() {
		BeforeEach(func() {
			agentAddr = ""
		})

		It("exits with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --agent-addr"))
		})
	})

	Context("when the --metrics-interval param is invalid", func() {
		BeforeEach(func() {
			metricsInterval = "10x"
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("invalid value \"10x\" for flag -metrics-interval"))
		})
	})

	Context("when the --ca param is not provided", func() {
		BeforeEach(func() {
			caPath = ""
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --ca"))
		})
	})

	Context("when the --cert param is not provided", func() {
		BeforeEach(func() {
			certPath = ""
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --cert"))
		})
	})

	Context("when the --key param is not provided", func() {
		BeforeEach(func() {
			keyPath = ""
		})

		It("returns with a non-zero exit code", func() {
			Eventually(session.ExitCode).Should(BeNumerically(">", 0))
		})

		It("provides a meaningful error message", func() {
			Eventually(func() *gbytes.Buffer {
				return session.Err
			}).Should(gbytes.Say("Must provide --key"))
		})
	})
})

type stubAgent struct {
	address   string
	lock      sync.Mutex
	envelopes []*loggregator_v2.Envelope
}

func newStubAgent() *stubAgent {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	agent := &stubAgent{
		address: l.Addr().String(),
	}

	tlsConfig, err := loggregator.NewIngressTLSConfig(
		Cert("loggregator-ca.crt"),
		Cert("metron.crt"),
		Cert("metron.key"),
	)
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)

	loggregator_v2.RegisterIngressServer(server, agent)
	go server.Serve(l)

	return agent
}

func (a *stubAgent) Sender(loggregator_v2.Ingress_SenderServer) error {
	panic("not implemented")
}

func (a *stubAgent) BatchSender(s loggregator_v2.Ingress_BatchSenderServer) error {
	for {
		if isDone(s.Context()) {
			return nil
		}

		eb, err := s.Recv()
		if err != nil {
			return err
		}

		a.lock.Lock()
		for _, e := range eb.GetBatch() {
			a.envelopes = append(a.envelopes, e)
		}
		a.lock.Unlock()
	}
}

func (a *stubAgent) Send(context.Context, *loggregator_v2.EnvelopeBatch) (*loggregator_v2.SendResponse, error) {
	panic("not implemented")
}

func (a *stubAgent) GetEnvelopes() []*loggregator_v2.Envelope {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.envelopes
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
