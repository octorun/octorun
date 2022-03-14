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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"octorun.github.io/octorun/util/remoteexec"
)

// RandomString returns a random alphanumeric string.
func RandomString(n int) string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyz"
	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	result := make([]byte, n)
	for i := range result {
		result[i] = charset[rnd.Intn(len(charset))]
	}
	return string(result)
}

func FindRunnerIDFromPod(pod *corev1.Pod, remoteexec remoteexec.RemoteExecutor) (int64, error) {
	var stdout, stderr bytes.Buffer
	command := []string{"bash", "-c", "cat .runner | jq .agentId | tr -d '\n'"}
	if err := remoteexec.Exec(pod, "runner", os.Stdin, &stdout, &stderr, command...); err != nil {
		return -1, err
	}

	if errs := strings.TrimSpace(stderr.String()); len(errs) > 0 {
		return -1, fmt.Errorf("error executing command: %s", errs)
	}

	runnerID, err := strconv.ParseInt(strings.TrimSpace(stdout.String()), 10, 0)
	if err != nil {
		return -1, err
	}

	return runnerID, nil
}
