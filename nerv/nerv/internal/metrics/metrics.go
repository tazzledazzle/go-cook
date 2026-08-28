// Package metrics defines the Prometheus collectors that make Nerv's
// own operation observable: how many projects get generated, by
// language, and how effective the dependency-resolution cache is.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics bundles every collector Nerv exposes. Construct once at
// startup and pass it (or its individual methods) to whichever
// components need to record events.
type Metrics struct {
	registry *prometheus.Registry

	projectsGeneratedTotal *prometheus.CounterVec
	resolutionCacheTotal   *prometheus.CounterVec
	renderDuration         prometheus.Histogram
}

// New builds a Metrics instance with its own registry (rather than the
// global default) so tests can spin up isolated instances without
// collisions.
func New() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		registry: reg,
		projectsGeneratedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nerv_projects_generated_total",
				Help: "Total number of projects generated, labeled by language.",
			},
			[]string{"language"},
		),
		resolutionCacheTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nerv_dependency_resolution_total",
				Help: "Total dependency resolutions, labeled by cache result (hit/miss).",
			},
			[]string{"result"},
		),
		renderDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "nerv_template_render_duration_seconds",
				Help:    "Time to render a template's files to disk.",
				Buckets: prometheus.DefBuckets,
			},
		),
	}

	reg.MustRegister(m.projectsGeneratedTotal, m.resolutionCacheTotal, m.renderDuration)

	return m
}

// RecordProjectGenerated increments the generated-projects counter for
// the given language.
func (m *Metrics) RecordProjectGenerated(language string) {
	m.projectsGeneratedTotal.WithLabelValues(language).Inc()
}

// RecordResolution increments the resolution counter, labeled "hit" or
// "miss" depending on whether the dependency cache was used.
func (m *Metrics) RecordResolution(cacheHit bool) {
	label := "miss"
	if cacheHit {
		label = "hit"
	}
	m.resolutionCacheTotal.WithLabelValues(label).Inc()
}

// ObserveRenderDuration records how long a template render took, in
// seconds.
func (m *Metrics) ObserveRenderDuration(seconds float64) {
	m.renderDuration.Observe(seconds)
}

// Handler returns an http.Handler serving this Metrics instance's data
// in Prometheus exposition format — mount at /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
