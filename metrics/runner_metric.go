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
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	"octorun.github.io/octorun/pkg/statemetrics"
)

type RunnerProvider struct{}

func (p *RunnerProvider) Namespace() string { return "octorun" }
func (p *RunnerProvider) Subsystem() string { return "runner" }
func (p *RunnerProvider) ProvideMetricFamily() []statemetrics.MetricFamily {
	return []statemetrics.MetricFamily{
		{
			Name: "labels",
			Help: "Kubernetes labels converted to Prometheus labels.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(r *octorunv1.Runner) []*statemetrics.Metric {
				k, v := MapToPrometheusLabels("label", r.Labels)
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
			MetricsFunc: p.WrapMetricsFunc(func(r *octorunv1.Runner) []*statemetrics.Metric {
				metrics := []*statemetrics.Metric{}
				if !r.CreationTimestamp.IsZero() {
					metrics = append(metrics, &statemetrics.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(r.CreationTimestamp.Unix()),
					})
				}

				return metrics
			}),
		},
		{
			Name: "status_phase",
			Help: "Current status phase.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(r *octorunv1.Runner) []*statemetrics.Metric {
				phase := r.Status.Phase
				if phase == "" {
					return []*statemetrics.Metric{}
				}

				phases := []struct {
					met      bool
					strphase string
				}{
					{phase == octorunv1.RunnerPendingPhase, string(octorunv1.RunnerPendingPhase)},
					{phase == octorunv1.RunnerIdlePhase, string(octorunv1.RunnerIdlePhase)},
					{phase == octorunv1.RunnerActivePhase, string(octorunv1.RunnerActivePhase)},
					{phase == octorunv1.RunnerCompletePhase, string(octorunv1.RunnerCompletePhase)},
				}

				metrics := make([]*statemetrics.Metric, len(phases))
				for i, p := range phases {
					metrics[i] = &statemetrics.Metric{

						LabelKeys:   []string{"phase"},
						LabelValues: []string{p.strphase},
						Value:       BoolFloat64(p.met),
					}
				}

				return metrics
			}),
		},
		{
			Name: "status_conditions",
			Help: "Current status conditions.",
			Type: prometheus.GaugeValue,
			MetricsFunc: p.WrapMetricsFunc(func(r *octorunv1.Runner) []*statemetrics.Metric {
				return MetaConditionsMetrics(r.Status.Conditions)
			}),
		},
	}
}

func (p *RunnerProvider) Lister(ctx context.Context, r client.Reader) cache.Lister {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			runnerList := &octorunv1.RunnerList{}
			err := r.List(ctx, runnerList, &client.ListOptions{Raw: &options})
			return runnerList, err
		},
	}
}

func (p *RunnerProvider) WrapMetricsFunc(fn func(*octorunv1.Runner) []*statemetrics.Metric) func(runtime.Object) []*statemetrics.Metric {
	return func(obj runtime.Object) []*statemetrics.Metric {
		runner := obj.(*octorunv1.Runner)
		metrics := fn(runner)
		for _, m := range metrics {
			m.LabelKeys = append([]string{"namespace", "runner"}, m.LabelKeys...)
			m.LabelValues = append([]string{runner.Namespace, runner.Name}, m.LabelValues...)
		}

		return metrics
	}
}
