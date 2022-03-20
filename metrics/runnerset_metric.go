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

package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/pkg/statemetrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RunnerSetProvider struct{}

func (p *RunnerSetProvider) Namespace() string { return "octorun" }
func (p *RunnerSetProvider) Subsystem() string { return "runnerset" }
func (p *RunnerSetProvider) ProvideMetricFamily() []statemetrics.MetricFamily {
	return []statemetrics.MetricFamily{
		{
			Name: "labels",
			Help: "Kubernetes labels converted to Prometheus labels.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				k, v := MapToPrometheusLabels("label", rs.Labels)
				return []*statemetrics.Metric{
					{
						LabelKeys:   k,
						LabelValues: v,
						Value:       1,
					},
				}
			}),
		},
		{
			Name: "created",
			Help: "Unix creation timestamp",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if !rs.CreationTimestamp.IsZero() {
					metrics = append(metrics, &statemetrics.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(rs.CreationTimestamp.Unix()),
					})
				}

				return metrics
			}),
		},
		{
			Name: "spec_runners",
			Help: "Number of desired runners.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if rs.Spec.Runners != nil {
					metrics = append(metrics, &statemetrics.Metric{
						Value: float64(*rs.Spec.Runners),
					})
				}

				return metrics
			}),
		},
		{
			Name: "status_runners",
			Help: "Most recently observed number of runners.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if rs.Spec.Runners != nil {
					metrics = append(metrics, &statemetrics.Metric{
						Value: float64(rs.Status.Runners),
					})
				}

				return metrics
			}),
		},
		{
			Name: "status_idle_runners",
			Help: "Most recently observed number of idle runners.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if rs.Spec.Runners != nil {
					metrics = append(metrics, &statemetrics.Metric{
						Value: float64(rs.Status.IdleRunners),
					})
				}

				return metrics
			}),
		},
		{
			Name: "status_active_runners",
			Help: "Most recently observed number of active runners.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if rs.Spec.Runners != nil {
					metrics = append(metrics, &statemetrics.Metric{
						Value: float64(rs.Status.ActiveRunners),
					})
				}

				return metrics
			}),
		},
		{
			Name: "status_conditions",
			Help: "Current status conditions.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				return MetaConditionsMetrics(rs.Status.Conditions)
			}),
		},
		{
			Name: "runner_active_ratio",
			Help: "Ratio of active runners per total runners.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(rs *octorunv1alpha1.RunnerSet) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				x := float64(rs.Status.ActiveRunners) / float64(rs.Status.Runners)
				v := int(x * 100)
				if rs.Spec.Runners != nil {
					metrics = append(metrics, &statemetrics.Metric{
						Value: float64(v) / 100,
					})
				}

				return metrics
			}),
		},
	}
}

func (p *RunnerSetProvider) Lister(ctx context.Context, r client.Reader) cache.Lister {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			runnersetList := &octorunv1alpha1.RunnerSetList{}
			err := r.List(ctx, runnersetList, &client.ListOptions{Raw: &options})
			return runnersetList, err
		},
	}
}

func (p *RunnerSetProvider) WrapMetricsFunc(fn func(*octorunv1alpha1.RunnerSet) []*statemetrics.Metric) func(runtime.Object) []*statemetrics.Metric {
	return func(obj runtime.Object) []*statemetrics.Metric {
		runnerset := obj.(*octorunv1alpha1.RunnerSet)
		metrics := fn(runnerset)
		for _, m := range metrics {
			m.LabelKeys = append([]string{"namespace", "runnerset"}, m.LabelKeys...)
			m.LabelValues = append([]string{runnerset.Namespace, runnerset.Name}, m.LabelValues...)
		}

		return metrics
	}
}
