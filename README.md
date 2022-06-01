# service-metrics-release

Sevice Metrics is a framework for easily sending metrics to
[Cloud Foundry's Loggregator](https://docs.cloudfoundry.org/loggregator/architecture.html)
system.

If you have any questions, or want to get attention for a PR or issue please reach out on the [#logging-and-metrics channel in the cloudfoundry slack](https://cloudfoundry.slack.com/archives/CUW93AF3M)

## User Documentation

User documentation can be found
[here](https://docs.pivotal.io/svc-sdk/service-metrics). Documentation is
targeted at service authors wishing to send metrics from their service and
operators wanting to configure service metrics.

## BOSH Release Artifacts

Service Metrics releases artifacts can be found on
[bosh.io](https://bosh.io/releases/github.com/cloudfoundry/service-metrics-release).
Service Metrics 1.5.6+ are licensed under Apache 2.0.

## Running local tests

1. `cd src/`
1. `./scripts/run-tests.sh`

## Deploying Service Metrics

1. Deploy this release with Loggregator components using a manifest similar to
   the one in `manifests/example_manifest.yml`. The example has comments to
   describe the necessary changes to variables.
1. See metrics with `cf tail` and `cf log-stream`. There should be a metric
   named `service_dummy` with a value of 99.
