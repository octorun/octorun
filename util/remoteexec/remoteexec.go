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

package remoteexec

import (
	"bytes"
	"io"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RemoteExecutor interface {
	Exec(obj client.Object, containerName string, in io.Reader, out, errout io.Writer, command ...string) error
}

type FakeRemoteExecutor struct {
	Out     *bytes.Buffer
	Errout  *bytes.Buffer
	Execerr error
}

func (re *FakeRemoteExecutor) Exec(obj client.Object, containerName string, in io.Reader, out, errout io.Writer, command ...string) error {
	_, _ = io.Copy(out, re.Out)
	_, _ = io.Copy(errout, re.Errout)
	return re.Execerr
}
