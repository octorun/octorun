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

package util

import (
	"bytes"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"octorun.github.io/octorun/util/remoteexec"
)

var executablePod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-pod",
		Namespace: "test-namespace",
	},
}

func TestFindRunnerIDFromPod(t *testing.T) {
	type args struct {
		pod        *corev1.Pod
		remoteexec remoteexec.RemoteExecutor
	}
	tests := []struct {
		name         string
		args         args
		wantRunnerID int64
		wantErr      bool
	}{
		{
			name: "remoteexec_has_out",
			args: args{
				pod: executablePod,
				remoteexec: &remoteexec.FakeRemoteExecutor{
					Out:     bytes.NewBufferString("1"),
					Errout:  bytes.NewBufferString(""),
					Execerr: nil,
				},
			},
			wantRunnerID: 1,
			wantErr:      false,
		},
		{
			name: "remoteexec_has_out_but_not_number",
			args: args{
				pod: executablePod,
				remoteexec: &remoteexec.FakeRemoteExecutor{
					Out:     bytes.NewBufferString("a"),
					Errout:  bytes.NewBufferString(""),
					Execerr: nil,
				},
			},
			wantRunnerID: -1,
			wantErr:      true,
		},
		{
			name: "remoteexec_has_errout",
			args: args{
				pod: executablePod,
				remoteexec: &remoteexec.FakeRemoteExecutor{
					Out:     bytes.NewBufferString("a"),
					Errout:  bytes.NewBufferString("error"),
					Execerr: nil,
				},
			},
			wantRunnerID: -1,
			wantErr:      true,
		},
		{
			name: "remoteexec_has_execerr",
			args: args{
				pod: executablePod,
				remoteexec: &remoteexec.FakeRemoteExecutor{
					Out:     bytes.NewBufferString("a"),
					Errout:  bytes.NewBufferString(""),
					Execerr: errors.New("foo"),
				},
			},
			wantRunnerID: -1,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRunnerID, err := FindRunnerIDFromPod(tt.args.pod, tt.args.remoteexec)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindRunnerIDFromPod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRunnerID != tt.wantRunnerID {
				t.Errorf("FindRunnerIDFromPod() = %v, want %v", gotRunnerID, tt.wantRunnerID)
			}
		})
	}
}
