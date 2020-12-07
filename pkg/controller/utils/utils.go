/*
Copyright 2020 The Crossplane Authors.

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

package utils

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// ExecIntoPod is used to select
func ExecIntoPod(client kubernetes.Interface, dpl *appsv1.Deployment, cmd string) error {
	command := []string{"/bin/bash", "-c", cmd}
	pod, err := getDeploymentPod(client, dpl)
	if err != nil {
		return err
	}
	if _, stderr, err := exec(client, command, pod, dpl.Namespace); err != nil {
		return errors.Wrapf(err, "failed to exec, %s", StringValueFallback(stderr, "no error message"))
	}
	return nil
}

// run exec command on pod
func exec(cs kubernetes.Interface, command []string, pod, ns string) (*string, *string, error) {
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(ns).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Command: command,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, scheme.ParameterCodec)

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return nil, nil, errors.Wrap(err, "error while creating executor")
	}

	var stdout, stderr bytes.Buffer
	var stdin io.Reader
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return String(stdout.String()), String(stderr.String()), err
	}

	return String(stdout.String()), String(stderr.String()), nil
}

func getDeploymentPod(cl kubernetes.Interface, dpl *appsv1.Deployment) (string, error) {
	name := dpl.Name
	ns := dpl.Namespace
	api := cl.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "deployment=" + name,
	}
	podList, err := api.Pods(ns).List(context.Background(), listOptions)
	if err != nil {
		return "", err
	}
	podListItems := podList.Items
	if len(podListItems) == 0 {
		return "", nil
	}
	return podListItems[0].Name, nil
}

// String is a utility function converting a string to a *string
func String(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringValue is a utility function converting a *string to a string
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// StringValueFallback is a utility function converting a converting a *string to a string with a fallback value
func StringValueFallback(s *string, fb string) string {
	if s == nil {
		return fb
	}
	return *s
}

// Int is a utility function converting an int to an int pointer
func Int(i int) *int {
	return &i
}

// IntValue is a utility function converting an *int32 to a value
func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Int32 is a utility function converting an int32 to a pointer
func Int32(i int32) *int32 {
	return &i
}
