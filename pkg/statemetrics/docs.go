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

// Package statemetrics is library contains interface and utilities to expose
// prometheus metrics for Kubernetes Resources state.
//
// The interface is inspired by how k8s.io/kube-state-metrics provides
// state metrics for Kubernetes Custom Resources. This library uses
// prometheus.Collector interface to write the metrics instead of using
// the own writer generator as k8s.io/kube-state-metrics do.
// That makes this library compatible with Prometheus Registry
package statemetrics
