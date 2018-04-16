package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"strings"

	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

const (
	stemcellAlias = "only-stemcell-alias"
	sourcePrefix  = "service-instance_"
)

func main() {
	serviceadapter.HandleCommandLineInvocation(
		os.Args,
		&ManifestGenerator{},
		&Binder{},
		&DashboardURLGenerator{},
	)
}

type ManifestGenerator struct{}

func (m *ManifestGenerator) GenerateManifest(
	serviceDeployment serviceadapter.ServiceDeployment,
	plan serviceadapter.Plan,
	requestParams serviceadapter.RequestParameters,
	previousManifest *bosh.BoshManifest,
	previousPlan *serviceadapter.Plan,
) (bosh.BoshManifest, error) {
	var releases []bosh.Release
	for _, r := range serviceDeployment.Releases {
		releases = append(releases, bosh.Release{
			Name:    r.Name,
			Version: r.Version,
		})
	}
	stemcells := []bosh.Stemcell{
		{
			Alias:   stemcellAlias,
			OS:      serviceDeployment.Stemcell.OS,
			Version: serviceDeployment.Stemcell.Version,
		},
	}

	sourceID := strings.TrimPrefix(serviceDeployment.DeploymentName, sourcePrefix)

	return bosh.BoshManifest{
		Name:      serviceDeployment.DeploymentName,
		Releases:  releases,
		Stemcells: stemcells,
		InstanceGroups: []bosh.InstanceGroup{
			{
				Name:      "service-metrics",
				Instances: 1,
				VMType:    "minimal",
				AZs:       []string{"z1"},
				Networks:  []bosh.Network{{Name: "default"}},
				Stemcell:  stemcellAlias,
				Jobs: []bosh.Job{
					{
						Name:    "service-metrics",
						Release: "service-metrics",
						Properties: map[string]interface{}{
							"service_metrics": map[string]interface{}{
								"origin":                     "service-metrics-injector",
								"source_id":                  sourceID,
								"execution_interval_seconds": 5,
								"metrics_command":            "/bin/echo",
								"metrics_command_args":       []string{"-n", buildMetrics()},
								"monit_dependencies":         []string{},
								"tls": map[string]interface{}{
									"ca":   plan.Properties["service_metrics_ca"],
									"cert": plan.Properties["service_metrics_cert"],
									"key":  plan.Properties["service_metrics_key"],
								},
							},
						},
					},
					{
						Name:    "loggregator_agent",
						Release: "loggregator-agent",
						Consumes: map[string]interface{}{
							"doppler": map[string]string{
								"from":       "doppler",
								"deployment": "cf",
							},
						},
						Properties: map[string]interface{}{
							"disable_udp": true,
							"bosh_dns":    true,
							"loggregator": map[string]interface{}{
								"tls": map[string]interface{}{
									"ca_cert": plan.Properties["loggregator_agent_ca"],
									"agent": map[string]interface{}{
										"cert": plan.Properties["loggregator_agent_cert"],
										"key":  plan.Properties["loggregator_agent_key"],
									},
								},
							},
						},
					},
					{
						Name:       "bosh-dns",
						Release:    "bosh-dns",
						Properties: map[string]interface{}{},
					},
				},
			},
		},
		Update: &bosh.Update{
			Canaries:        10,
			MaxInFlight:     10,
			CanaryWatchTime: "30000 - 60000",
			UpdateWatchTime: "5000 - 60000",
			Serial: func() *bool {
				t := true
				return &t
			}(),
		},
	}, nil
}

type Binder struct{}

func (b *Binder) CreateBinding(
	bindingID string,
	deploymentTopology bosh.BoshVMs,
	manifest bosh.BoshManifest,
	requestParams serviceadapter.RequestParameters,
) (serviceadapter.Binding, error) {
	return serviceadapter.Binding{}, nil
}

func (b *Binder) DeleteBinding(
	bindingID string,
	deploymentTopology bosh.BoshVMs,
	manifest bosh.BoshManifest,
	requestParams serviceadapter.RequestParameters,
) error {
	return nil
}

type DashboardURLGenerator struct{}

func (d *DashboardURLGenerator) DashboardUrl(
	instanceID string,
	plan serviceadapter.Plan,
	manifest bosh.BoshManifest,
) (serviceadapter.DashboardUrl, error) {
	return serviceadapter.DashboardUrl{
		DashboardUrl: "https://service-metrics.coconut.cf-app.com",
	}, nil
}

func buildMetrics() string {
	requests := rand.Int() % 1000
	cpu := rand.Int() % 100
	m := []map[string]interface{}{
		map[string]interface{}{
			"name":  "http_failures",
			"delta": 1,
		},
		map[string]interface{}{
			"name":  "http_requests",
			"delta": requests,
		},
		map[string]interface{}{
			"key":   "cpu_usage",
			"value": cpu,
			"unit":  "percent",
		},
	}

	data, err := json.Marshal(&m)
	if err != nil {
		panic(err)
	}

	return string(data)
}
