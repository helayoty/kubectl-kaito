package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type LogsOptions struct {
	configFlags   *genericclioptions.ConfigFlags
	WorkspaceName string
	Namespace     string
	Follow        bool
	Tail          int64
	Container     string
}

func NewLogsCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &LogsOptions{
		configFlags: configFlags,
		Tail:        -1,
	}

	cmd := &cobra.Command{
		Use:   "logs <workspace-name>",
		Short: "Get logs from Kaito workspace pods",
		Long: `Get logs from Kaito workspace pods.

This command retrieves logs from the pods associated with a Kaito workspace,
which can help with debugging inference or fine-tuning issues.`,
		Example: `  # Get logs from workspace pods
  kubectl kaito logs workspace-llama-3
  
  # Follow logs (stream)
  kubectl kaito logs workspace-llama-3 --follow
  
  # Get last 100 lines
  kubectl kaito logs workspace-llama-3 --tail 100
  
  # Get logs from specific container
  kubectl kaito logs workspace-llama-3 --container inference`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("workspace name is required")
			}
			o.WorkspaceName = args[0]

			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&o.Follow, "follow", "f", false, "Follow log output")
	cmd.Flags().Int64Var(&o.Tail, "tail", -1, "Number of lines to show from the end of the logs")
	cmd.Flags().StringVarP(&o.Container, "container", "c", "", "Container name")

	return cmd
}

func (o *LogsOptions) Complete() error {
	// Get namespace from flags or config
	if o.Namespace == "" {
		if o.configFlags.Namespace != nil && *o.configFlags.Namespace != "" {
			o.Namespace = *o.configFlags.Namespace
		} else {
			o.Namespace = "default"
		}
	}
	return nil
}

func (o *LogsOptions) Validate() error {
	if o.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required")
	}
	return nil
}

func (o *LogsOptions) Run() error {
	// Get REST config
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Find pods with the workspace label
	labelSelector := fmt.Sprintf("app=%s", o.WorkspaceName)
	pods, err := clientset.CoreV1().Pods(o.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		// Try alternative label selector patterns
		altSelectors := []string{
			fmt.Sprintf("workspace=%s", o.WorkspaceName),
			fmt.Sprintf("kaito.sh/workspace=%s", o.WorkspaceName),
		}

		for _, selector := range altSelectors {
			pods, err = clientset.CoreV1().Pods(o.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: selector,
			})
			if err == nil && len(pods.Items) > 0 {
				break
			}
		}

		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found for workspace %s in namespace %s", o.WorkspaceName, o.Namespace)
		}
	}

	// If multiple pods, show logs from all of them
	for _, pod := range pods.Items {
		if len(pods.Items) > 1 {
			fmt.Printf("==> Pod: %s <==\n", pod.Name)
		}

		containerName := o.Container
		if containerName == "" {
			// Use the first container if not specified
			if len(pod.Spec.Containers) > 0 {
				containerName = pod.Spec.Containers[0].Name
			}
		}

		err := o.streamLogs(clientset, pod.Name, containerName)
		if err != nil {
			fmt.Printf("Error getting logs from pod %s: %v\n", pod.Name, err)
			continue
		}

		if len(pods.Items) > 1 {
			fmt.Println()
		}
	}

	return nil
}

func (o *LogsOptions) streamLogs(clientset kubernetes.Interface, podName, containerName string) error {
	logOptions := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    o.Follow,
	}

	if o.Tail >= 0 {
		logOptions.TailLines = &o.Tail
	}

	req := clientset.CoreV1().Pods(o.Namespace).GetLogs(podName, logOptions)

	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}
	defer podLogs.Close()

	// Copy logs to stdout
	_, err = io.Copy(os.Stdout, podLogs)
	if err != nil {
		return fmt.Errorf("failed to copy logs: %w", err)
	}

	return nil
}
