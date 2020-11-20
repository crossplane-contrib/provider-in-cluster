package utils

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

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

func getDeploymentPod(cl kubernetes.Interface, dpl *appsv1.Deployment) (podName string, err error) {
	name := dpl.Name
	ns := dpl.Namespace
	api := cl.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "deployment=" + name,
	}
	podList, _ := api.Pods(ns).List(context.Background(), listOptions)
	podListItems := podList.Items
	if len(podListItems) == 0 {
		return "", err
	}
	podName = podListItems[0].Name
	return podName, nil
}

func String(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func StringValueFallback(s *string, fb string) string {
	if s == nil {
		return fb
	}
	return *s
}

func Int(i int) *int {
	return &i
}

func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
