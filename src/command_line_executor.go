package main

import (
	"os"
	"os/exec"
	"syscall"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/service-metrics/metrics"
)

// CommandLineExecutor implements metrics.Executor
type CommandLineExecutor struct {
	logger metrics.Logger
}

func NewCommandLineExecutor(l metrics.Logger) CommandLineExecutor {
	return CommandLineExecutor{
		logger: l,
	}
}

func (e CommandLineExecutor) Run(c *exec.Cmd) ([]byte, error) {
	action := "executing-metrics-cmd"

	e.logger.Info(action, lager.Data{
		"event": "starting",
	})

	out, err := c.CombinedOutput()

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			e.logger.Error(action, err, lager.Data{
				"event":  "failed",
				"output": "no metrics command has been configured, cannot collect metrics",
			})
			os.Exit(1)
		}

		exitStatus := c.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
		if exitStatus == 10 {
			e.logger.Info(action, lager.Data{
				"event":  "not yet ready to emit metrics",
				"output": string(out),
			})
			return nil, err
		}

		e.logger.Error(action, err, lager.Data{
			"event":  "failed",
			"output": string(out),
		})
		os.Exit(0)
	}

	e.logger.Info(action, lager.Data{
		"event": "done",
	})

	return out, nil
}
