package main

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"ukho.gov.uk/azdo-build-exporter/azdo"
)

type azDoCollector struct {
	AzDoClient *azdo.AzDoClient
	lastScrape time.Time

	mu     sync.RWMutex
	cached []prometheus.Metric
}

func newAzDoCollector(az azdo.AzDoClient) *azDoCollector {
	return &azDoCollector{AzDoClient: &az}
}

// startRefresh populates the metrics cache immediately and then on every interval tick.
func (azc *azDoCollector) startRefresh(interval time.Duration) {
	go func() {
		azc.refresh()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			azc.refresh()
		}
	}()
}

func (azc *azDoCollector) refresh() {
	start := time.Now()
	logger := log.WithFields(log.Fields{"serverName": azc.AzDoClient.Name})

	projects, err := azc.AzDoClient.GetProjects()
	if err != nil {
		logger.WithField("error", err).Error("Failed to retrieve projects, keeping cached metrics")
		return
	}

	logger.Info("Retrieved Projects")

	chanBuilds, errOccurred := azc.scrapeBuilds(projects)
	if errOccurred {
		return
	}

	var metrics []prometheus.Metric
	for metric := range azc.calculateMetrics(chanBuilds) {
		metrics = append(metrics, metric)
	}

	metrics = append(metrics, prometheus.MustNewConstMetric(
		totalcollectDurationDesc,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	))

	azc.mu.Lock()
	azc.cached = metrics
	azc.mu.Unlock()

	azc.lastScrape = start
	logger.WithField("duration", time.Since(start)).Info("Metrics cache refreshed")
}

func (azc *azDoCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(azc, ch)
}

func (azc *azDoCollector) Collect(publishMetrics chan<- prometheus.Metric) {
	azc.mu.RLock()
	defer azc.mu.RUnlock()

	for _, metric := range azc.cached {
		publishMetrics <- metric
	}
}

func (azc *azDoCollector) scrapeBuilds(projects []azdo.Project) (<-chan metricsContext,bool) {

	metrics := make(chan metricsContext)
	errOccurred := false

	log.WithFields(log.Fields{"serverName": azc.AzDoClient.Name}).Info("calculateMetrics")
	var wg sync.WaitGroup

	for _, project := range projects {
		wg.Add(1)
		go func(p azdo.Project) {
			log.Info(p.Name)
			finishedBuilds,currentBuilds, err := azc.AzDoClient.GetBuilds(p.Name,azc.lastScrape)
			if err != nil {
				errOccurred = true
			}

			metrics <- metricsContext{Project:p, Builds: finishedBuilds, Current: currentBuilds}
			wg.Done()
		}(project)
	}

	go func() {
		wg.Wait()
		close(metrics)
	}()
	return metrics, errOccurred
}

func (azc *azDoCollector) calculateMetrics(metricsContextChanIn <-chan metricsContext) <-chan prometheus.Metric {
	metrics := make(chan prometheus.Metric)

	go func() {
		for mc := range metricsContextChanIn {
			
			buildMetrics := calculateBuildMetrics(mc)

			for _, buildMetric := range buildMetrics {
				metrics <- buildMetric
			}
		}
		close(metrics)
	}()
	return metrics
}

// Contains all the information needed to calculate the metrics
type metricsContext struct {
	Project azdo.Project
	Builds   []azdo.Build
	Current []azdo.Build
}
