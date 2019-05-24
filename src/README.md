# service-metrics

Sevice Metrics is a framework for easily sending metrics to [Cloud Foundry's Loggregator](https://github.com/cloudfoundry/loggregator) system.

In production, this application is deployed via a [BOSH release](https://github.com/cloudfoundry/service-metrics-release). See its repo for more details.

## User Documentation

User documentation can be found [here](https://docs.pivotal.io/svc-sdk/service-metrics). Documentation is targeted at service authors wishing to deploy their services on-demand and operators wanting to offer services on-demand.

## Useage

Takes a 'metrics command' as input, runs the command, and forwards the resulting metrics output to metron.

## Running the tests

`./scripts/run-tests.sh`
