package main

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"./azdo"
)

type azDoCollector struct {
	AzDoClient *azdo.AzDoClient
	lastScrape time.Time
}

func newAzDoCollector(az azdo.AzDoClient) *azDoCollector {
	return &azDoCollector{AzDoClient: &az}
}

func (azc *azDoCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(azc, ch)
}

func (azc *azDoCollector) Collect(publishMetrics chan<- prometheus.Metric) {
	
	start := time.Now()
	
	projects,err := azc.AzDoClient.GetProjects()

	if err != nil {
		return
	}

	log.WithFields(log.Fields{"serverName": azc.AzDoClient.Name}).Info("Retrieved Projects")


	chanBuilds, errOccurred := azc.scrapeBuilds(projects)
	if errOccurred {
		return
	}
	
	chanCalculatedMetrics := azc.calculateMetrics(chanBuilds)

	//Publish the buffered metrics
	for metric := range chanCalculatedMetrics {
		publishMetrics <- metric
	}

	publishMetrics <- prometheus.MustNewConstMetric(
		totalcollectDurationDesc,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)

	azc.lastScrape = time.Now()
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
			log.Info(len(finishedBuilds))
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
