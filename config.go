package main

import (
	"net/url"

	"ukho.gov.uk/azdo-build-exporter/azdo"
)

var (
	portDefault     = 8080
	endpointDefault = "/metrics"
)

type config struct {
	Servers  map[string]azDoConfig
	Proxy    proxy
	Exporter exporter
}

type exporter struct {
	Port     int
	Endpoint string
}

type proxy struct {
	URL      string
	proxyURL *url.URL
}

type azDoConfig struct {
	azdo.AzDoClient
	UseProxy bool
}
