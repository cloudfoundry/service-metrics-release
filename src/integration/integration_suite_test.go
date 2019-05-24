package integration_test

import (
	"log"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var execPath string

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	srcPath := "code.cloudfoundry.org/service-metrics"
	var err error
	execPath, err = gexec.Build(srcPath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", srcPath, err)
	}
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func runCmd(
	origin string,
	sourceID string,
	debugLog bool,
	agentAddress string,
	metricsInterval string,
	metricsCmd string,
	caPath string,
	certPath string,
	keyPath string,
	metricsCmdArgs []string,
	envVars []string,
) *gexec.Session {
	var cmdArgs []string
	if origin != "" {
		cmdArgs = append(cmdArgs, "--origin", origin)
	}

	if sourceID != "" {
		cmdArgs = append(cmdArgs, "--source-id", sourceID)
	}

	if agentAddress != "" {
		cmdArgs = append(cmdArgs, "--agent-addr", agentAddress)
	}

	if metricsInterval != "" {
		cmdArgs = append(cmdArgs, "--metrics-interval", metricsInterval)
	}

	if metricsCmd != "" {
		cmdArgs = append(cmdArgs, "--metrics-cmd", metricsCmd)
	}

	if caPath != "" {
		cmdArgs = append(cmdArgs, "--ca", caPath)
	}

	if certPath != "" {
		cmdArgs = append(cmdArgs, "--cert", certPath)
	}

	if keyPath != "" {
		cmdArgs = append(cmdArgs, "--key", keyPath)
	}

	if debugLog {
		cmdArgs = append(cmdArgs, "--debug")
	}

	for _, arg := range metricsCmdArgs {
		cmdArgs = append(cmdArgs, "--metrics-cmd-arg", arg)
	}

	cmd := exec.Command(execPath, cmdArgs...)

	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	cmd.Env = envVars

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}
