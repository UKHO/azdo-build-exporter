package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

var (
	totalcollectDurationDesc = prometheus.NewDesc(
		"azdo_build_build_total_scrape_duration_seconds",
		"Duration of time it took to scrape total of builds",
		[]string{},
		nil,
	)

	buildTimeToCompleteDesc = prometheus.NewDesc(
		"azdo_build_time_in_seconds",
		"Build time in seconds",
		[]string{"Project", "BuildId", "BuildNumber", "DefinitionId", "DefinitionName","status","result"},
		nil,
	)

	totalJobsDesc = prometheus.NewDesc(
		"azdo_build_count",
		"Total of builds for project",
		[]string{"project"},
		nil,
	)

	queuedJobsDesc = prometheus.NewDesc(
		"azdo_build_queued_count",
		"Total of queued builds for project",
		[]string{"project"},
		nil,
	)

	runningJobsDesc = prometheus.NewDesc(
		"azdo_build_running_jobs",
		"Total of running builds for project",
		[]string{"project"},
		nil,
	)

)

func calculateBuckets() []float64 {
	var b = buckets(0, 15, 8)                       // start at 0, gap of 15 between buckets and 10 of them
	b = append(b, buckets(b[len(b)-1], 30, 10)...)  // start of the last value of previous slice, gap of 30 between buckets and 10 of them
	b = append(b, buckets(b[len(b)-1], 60, 28)...)  // start of the last value of previous slice, gap of 30 between buckets and 10 of them
	b = append(b, buckets(b[len(b)-1], 300, 11)...) // start of the last value of previous slice, gap of 300 between buckets and 11 of them
	return b
}

func buckets(start float64, gap float64, count int) []float64 {
	var s []float64
	var currentBucket = start

	for i := 0; i < count; i++ {
		currentBucket = currentBucket + gap
		s = append(s, currentBucket)
	}

	return s
}

func calculateBuildMetrics(mc metricsContext) []prometheus.Metric {

	promMetrics := []prometheus.Metric{}

	for _,build := range mc.Builds {
		var buildTime = build.FinishTime.Sub(build.StartTime)
		createMetric := prometheus.MustNewConstMetric(
		buildTimeToCompleteDesc,
					prometheus.GaugeValue,
					float64(buildTime.Seconds()),
					mc.Project.Name,
					strconv.Itoa(build.Id),
					build.Number,
					strconv.Itoa(build.Definition.Id),
					build.Definition.Name,
					build.Status,
					build.Result,
				)

				promMetrics = append(promMetrics, createMetric)
			}

			promMetrics = append(promMetrics, calculateHistograms(mc)...)

	return promMetrics
}

func calculateHistograms(metricContext metricsContext) []prometheus.Metric {

	totalTimes := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        "azdo_build_total_length_secs",
		Help:        "Total length of azdo_build duration for pool",
		Buckets:     calculateBuckets(),
		ConstLabels: map[string]string{"Project": metricContext.Project.Name},
	})

	for _, job := range metricContext.Builds {
		totalTime := job.FinishTime.Sub(job.QueueTime)
		totalTimes.Observe(totalTime.Seconds())
	}

	queueTimes := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        "azdo_build_queue_length_secs",
		Help:        "Total length of queue duration for build",
		Buckets:     prometheus.ExponentialBuckets(1, 2, 10), // 10 buckets, starting at one, doubling
		ConstLabels: map[string]string{"project": metricContext.Project.Name},
	})

	for _, job := range metricContext.Builds {
		queueTime := job.StartTime.Sub(job.QueueTime) // Time received by the agent - Time queued by the user
		queueTimes.Observe(queueTime.Seconds())
	}

	jobTimes := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        "azdo_build_running_length_secs",
		Help:        "Total length of queue duration for pool",
		Buckets:     calculateBuckets(),
		ConstLabels: map[string]string{"project": metricContext.Project.Name},
	})

	for _, job := range metricContext.Builds {
		jobTime := job.FinishTime.Sub(job.StartTime)
		jobTimes.Observe(jobTime.Seconds())
	}
	return []prometheus.Metric{
		totalTimes,
		queueTimes,
		jobTimes,
	}
}

func calculateQueueMetrics(metricContext metricsContext) []prometheus.Metric {

	queuedTotal := 0
	runningTotal := 0

	for _, currentBuild := range metricContext.Current {
		if currentBuild.StartTime.IsZero() { //Then the job hasn't started and is therefore queued
			queuedTotal++
		} else {
			runningTotal++
		}
	}

	calculatedMetrics := []prometheus.Metric{
		prometheus.MustNewConstMetric(
			totalJobsDesc,
			prometheus.GaugeValue,
			float64(len(metricContext.Current)),
			metricContext.Project.Name,
		),
		prometheus.MustNewConstMetric(
			runningJobsDesc,
			prometheus.GaugeValue,
			float64(runningTotal),
			metricContext.Project.Name,
		),
		prometheus.MustNewConstMetric(
			queuedJobsDesc,
			prometheus.GaugeValue,
			float64(queuedTotal),
			metricContext.Project.Name,
		),
	}

	return calculatedMetrics

}

