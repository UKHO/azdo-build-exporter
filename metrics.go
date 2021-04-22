package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

var (
	totalcollectDurationDesc = prometheus.NewDesc(
		"azdo_build_build_total_scrape_duration_seconds",
		"Duration of time it took to scrape total of builds",
		[]string{},
		nil,
	)

	buildTimeToCompleteDesc = prometheus.NewDesc(
		"azdo_build_complete_in_seconds",
		"Build complete in seconds",
		[]string{"Project", "BuildId", "BuildNumber", "DefinitionId", "DefinitionName","status","result"},
		nil,
	)

	buildTimeQueuedDesc = prometheus.NewDesc(
		"azdo_build_queued_in_seconds",
		"Build queued in seconds",
		[]string{"Project", "BuildId", "BuildNumber", "DefinitionId", "DefinitionName","status","result"},
		nil,
	)

	buildTimeRunningDesc = prometheus.NewDesc(
		"azdo_build_running_in_seconds",
		"Build running in seconds",
		[]string{"Project", "BuildId", "BuildNumber", "DefinitionId", "DefinitionName","status","result"},
		nil,
	)

	buildTotalDesc = prometheus.NewDesc(
		"azdo_build_count",
		"Total of builds for project",
		[]string{"project"},
		nil,
	)

	buildResultSuccessDesc = prometheus.NewDesc(
		"azdo_build_result_success_count",
		"Build Result Success",
		[]string{"Project","DefinitionId","DefinitionName"},
		nil,
	)

	buildResultFailDesc = prometheus.NewDesc(
		"azdo_build_result_failed_count",
		"Build Result Failed",
		[]string{"Project","DefinitionId","DefinitionName"},
		nil,
	)

	buildResultCancelledDesc = prometheus.NewDesc(
		"azdo_build_result_cancelled_count",
		"Build Result Cancelled",
		[]string{"Project","DefinitionId","DefinitionName"},
		nil,
	)

	queuedJobsDesc = prometheus.NewDesc(
		"azdo_build_queued_count",
		"Total of queued builds for project",
		[]string{"project"},
		nil,
	)

	runningJobsDesc = prometheus.NewDesc(
		"azdo_build_running_count",
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

	for _,build := range mc.Current {
		if(build.StartTime.IsZero()) {
			var queueTime = time.Now().Sub(build.QueueTime)
			createMetric := prometheus.MustNewConstMetric(
				buildTimeQueuedDesc,
				prometheus.GaugeValue,
				float64(queueTime.Seconds()),
				mc.Project.Name,
				strconv.Itoa(build.Id),
				build.Number,
				strconv.Itoa(build.Definition.Id),
				build.Definition.Name,
				build.Status,
				build.Result,
			)
			promMetrics = append(promMetrics, createMetric)
		} else {
			var buildTime = time.Now().Sub(build.StartTime)
			createMetric := prometheus.MustNewConstMetric(
				buildTimeRunningDesc,
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
	}

	promMetrics = append(promMetrics, calculateHistograms(mc)...)
	promMetrics = append(promMetrics, calculateQueueMetrics(mc)...)
	promMetrics = append(promMetrics, calculateBuildResultMetrics(mc)...)

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
			buildTotalDesc,
			prometheus.GaugeValue,
			float64(len(metricContext.Current) + len(metricContext.Builds)),
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

func calculateBuildResultMetrics(metricContext metricsContext) []prometheus.Metric {

	type buildResultMetric struct {
		Project string
		DefinitionId int
		DefinitionName string
		Succeeded int
		Failed int
		Cancelled int
	}

	m := make(map[string]buildResultMetric)

	for _, build := range metricContext.Builds {
		var buildResult = metricContext.Project.Name + strconv.Itoa(build.Definition.Id)
		metric, ok := m[buildResult]
		if ok {
			if(build.Result == "succeeded") {
				metric.Succeeded++
			}
			if(build.Result == "failed") {
				metric.Failed++
			}
			if(build.Result == "cancelled") {
				metric.Cancelled++
			}
		} else {
			if(build.Result == "succeeded") {
			metric = buildResultMetric{Succeeded: 1, Failed: 0, Cancelled: 0, Project: metricContext.Project.Name, DefinitionId: build.Definition.Id, DefinitionName: build.Definition.Name}
			}
			if(build.Result == "failed") {
			metric = buildResultMetric{Succeeded: 0, Failed: 1, Cancelled: 0, Project: metricContext.Project.Name, DefinitionId: build.Definition.Id, DefinitionName: build.Definition.Name}
			}
			if(build.Result == "cancelled") {
			metric = buildResultMetric{Succeeded: 0, Failed: 0,Cancelled:1, Project: metricContext.Project.Name, DefinitionId: build.Definition.Id, DefinitionName: build.Definition.Name}
			}
		}

		m[buildResult] = metric
	}

	promMetrics := []prometheus.Metric{}
	for _, p := range m {

		promMetric := prometheus.MustNewConstMetric(
			buildResultSuccessDesc,
			prometheus.GaugeValue,
			float64(p.Succeeded),
			p.Project, strconv.Itoa(p.DefinitionId), p.DefinitionName)

		promMetrics = append(promMetrics, promMetric)

		promFailMetric := prometheus.MustNewConstMetric(
			buildResultFailDesc,
			prometheus.GaugeValue,
			float64(p.Failed),
			p.Project, strconv.Itoa(p.DefinitionId), p.DefinitionName)

		promMetrics = append(promMetrics, promFailMetric)

		promCancelMetric := prometheus.MustNewConstMetric(
			buildResultCancelledDesc,
			prometheus.GaugeValue,
			float64(p.Cancelled),
			p.Project, strconv.Itoa(p.DefinitionId), p.DefinitionName)

		promMetrics = append(promMetrics, promCancelMetric)
	}
	return promMetrics
}

