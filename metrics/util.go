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
	"regexp"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"octorun.github.io/octorun/pkg/statemetrics"
)

var (
	conditionStatuses = []metav1.ConditionStatus{metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionUnknown}
)

// BoolFloat64 returns 1 if true and 0
// otherwise from given boolean value.
func BoolFloat64(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

// MetaConditionsMetrics returns metrics from given slice of metav1.Condition
func MetaConditionsMetrics(conditions []metav1.Condition) []*statemetrics.Metric {
	metrics := make([]*statemetrics.Metric, len(conditions)*len(conditionStatuses))
	for i, c := range conditions {
		conditionMetrics := make([]*statemetrics.Metric, len(conditionStatuses))
		for i, status := range conditionStatuses {
			conditionMetrics[i] = &statemetrics.Metric{
				LabelValues: []string{strings.ToLower(string(status))},
				Value:       BoolFloat64(c.Status == status),
			}
		}

		for j, cm := range conditionMetrics {
			selectedMetric := cm
			selectedMetric.LabelKeys = []string{"condition", "status"}
			selectedMetric.LabelValues = append([]string{c.Type}, selectedMetric.LabelValues...)
			metrics[i*len(conditionStatuses)+j] = selectedMetric
		}
	}

	return metrics
}

// MapToKeysValuesLabels returns slice of keys and values prometheus labels.
func MapToPrometheusLabels(prefix string, m map[string]string) ([]string, []string) {
	keys := make([]string, 0, len(m))
	values := make([]string, 0, len(m))
	sortedKeys := make([]string, 0)
	for k := range m {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		key := normalizeLabelName(prefix, k)
		keys = append(keys, key)
		values = append(values, m[k])
	}

	return keys, values
}

func sanitizeLabelName(s string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "_")
}

func normalizeLabelName(prefix, s string) string {
	sanitized := sanitizeLabelName(s)
	snake := regexp.MustCompile("([a-z0-9])([A-Z])").ReplaceAllString(sanitized, "${1}_${2}")
	return prefix + "_" + strings.ToLower(snake)
}
