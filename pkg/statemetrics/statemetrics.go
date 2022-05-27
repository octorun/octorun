/*
Copyright 2022 The Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package statemetrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("statemetrics")

// Metric represents a single time series.
type Metric struct {
	// LabelKeys list of prometheus label keys.
	LabelKeys []string

	// LabelValues list of prometheus label values.
	LabelValues []string

	// Value is prometheus metric value.
	Value float64
}

// MetricFamily represents a set of metrics with the same name and help text.
type MetricFamily struct {
	// Name is name of metrics
	Name string

	// Help provides information about this metric.
	Help string

	// Type is prometheus metric types.
	Type prometheus.ValueType

	// MetricsFunc knowns how to provide the metrics from given Kubernetes Object.
	MetricsFunc func(obj runtime.Object) []*Metric
}

// Provider knowns how to construct and scrape the metrics.
type Provider interface {
	// Namespace returns the prometheus namespace of this provider.
	Namespace() string

	// Subsystem returns the prometheus subsystem of this provider.
	Subsystem() string

	// ProvideMetricFamily returns slice of MetricFamily.
	ProvideMetricFamily() []MetricFamily

	// Lister knowns how to fetch list of Kubernetes Objects.
	Lister(ctx context.Context, r client.Reader) cache.Lister
}

// Collector collects prometheus metrics from given providers
type Collector struct {
	// APIReader is a controller-runtime client.Reader
	// that configured to uses the API server instead of cache.
	APIReader client.Reader

	// CacheReader is a controller-runtime client.Reader
	// that configured to uses controller-runtime cache.
	CacheReader client.Reader

	// providers holds registered metric providers.
	providers []Provider
}

func NewCollector(mgr ctrl.Manager, providers ...Provider) *Collector {
	return &Collector{
		APIReader:   mgr.GetAPIReader(),
		CacheReader: mgr.GetCache(),
		providers:   providers,
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	for _, provider := range c.providers {
		for _, metric := range c.Scrape(ctx, c.CacheReader, provider) {
			ch <- metric
		}
	}
}

// Scrape scrapes the metrics from the given provider. The reader
// may be given from either APIReader or CacheReader.
func (c *Collector) Scrape(ctx context.Context, reader client.Reader, provider Provider) []prometheus.Metric {
	var metrics []prometheus.Metric
	objList, err := provider.Lister(ctx, reader).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "unable to fetch Resources")
		// TODO: collect fail scrape metric instead
		return nil
	}

	objs, err := meta.ExtractList(objList)
	if err != nil {
		log.Error(err, "unable to fetch Resources")
		// TODO: collect fail scrape metric instead
		return nil
	}

	for _, p := range provider.ProvideMetricFamily() {
		metricName := prometheus.BuildFQName(provider.Namespace(), provider.Subsystem(), p.Name)
		for _, obj := range objs {
			for _, m := range p.MetricsFunc(obj) {
				metrics = append(metrics, prometheus.MustNewConstMetric(
					prometheus.NewDesc(metricName, p.Help, m.LabelKeys, nil),
					p.Type,
					m.Value,
					m.LabelValues...,
				))
			}
		}
	}

	return metrics
}
