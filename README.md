# service-metrics-release

A BOSH release for a [Service Metrics](https://github.com/cloudfoundry/service-metrics).

Sevice Metrics is a framework for easily sending metrics to [Cloud Foundry's Loggregator](https://github.com/cloudfoundry/loggregator) system.

## User Documentation

User documentation can be found [here](https://docs.pivotal.io/svc-sdk/service-metrics). Documentation is targeted at service authors wishing to send metrics from their service and operators wanting to configure service metrics.

## BOSH Release Artifacts

Service Metrics releases artifacts can be found on [PivNet](https://network.pivotal.io/products/service-metrics-sdk). Service Metrics 1.5.6+ are licensed under Apache 2.0.

## Running the system tests

1. Deploy this release (and metron_agent) using a manifest similar to the one in `manifests/example_manifest.yml`.
1. `cp .envrc.template .envrc`
1. Fill in the appropriate values for your environment
1. `go get github.com/cloudfoundry/noaa/samples/firehose`
1. `bundle install`
1. `bundle exec rake spec`
