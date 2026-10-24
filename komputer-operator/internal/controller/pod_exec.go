package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExecer runs a command in a pod container and streams its stdout.
type PodExecer interface {
	ExecStream(ctx context.Context, ns, pod, container string, cmd []string, stdout io.Writer) error
}

// remotecommandExecer is the PodExecer backed by the pods/exec subresource.
type remotecommandExecer struct {
	config    *rest.Config
	clientset kubernetes.Interface
}

// NewPodExecer builds the default PodExecer from the manager's REST config.
func NewPodExecer(config *rest.Config) (PodExecer, error) {
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &remotecommandExecer{config: config, clientset: cs}, nil
}

func (e *remotecommandExecer) ExecStream(ctx context.Context, ns, pod, container string, cmd []string, stdout io.Writer) error {
	req := e.clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(ns).Name(pod).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(e.config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("build exec: %w", err)
	}
	var stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: stdout, Stderr: &stderr}); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, bytes.TrimSpace(stderr.Bytes()))
		}
		return err
	}
	return nil
}
