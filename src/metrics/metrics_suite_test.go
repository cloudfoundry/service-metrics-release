package metrics_test

import (
	"testing"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

type spyLogger struct {
	// map of action strings to slice of data called against the actions
	infoKey   string
	infoData  []lager.Data
	errAction string
	errData   []lager.Data
	err       error
	errCalled bool
}

func (l *spyLogger) Info(action string, data ...lager.Data) {
	l.infoKey = action
	l.infoData = data
}

func (l *spyLogger) Error(action string, err error, data ...lager.Data) {
	l.errAction = action
	l.errData = data
	l.err = err
	l.errCalled = true
}
