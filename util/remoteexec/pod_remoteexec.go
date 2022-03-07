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
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type PodRemoteExecutor struct {
	Config *rest.Config
	Scheme *runtime.Scheme
}

func (re *PodRemoteExecutor) Exec(obj client.Object, containerName string, in io.Reader, out, errout io.Writer, command ...string) error {
	gvk, err := apiutil.GVKForObject(obj, re.Scheme)
	if err != nil {
		return err
	}

	rest, err := apiutil.RESTClientForGVK(gvk, false, re.Config, serializer.NewCodecFactory(re.Scheme))
	if err != nil {
		return err
	}

	req := rest.Post().
		Name(obj.GetName()).
		Namespace(obj.GetNamespace()).
		Resource("pods").
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     in != nil,
			Stdout:    out != nil,
			Stderr:    errout != nil,
		}, runtime.NewParameterCodec(re.Scheme))
	exec, err := remotecommand.NewSPDYExecutor(re.Config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  in,
		Stdout: out,
		Stderr: errout,
		Tty:    false,
	})
}
