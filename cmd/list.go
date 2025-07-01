package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig string
	namespace  string
)

// Main commands
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes resources",
	Long:  "List various Kubernetes resources like deployments and pods",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create Kubernetes resources",
	Long:  "Create various Kubernetes resources like deployments and pods",
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete Kubernetes resources",
	Long:  "Delete various Kubernetes resources like deployments and pods",
}

// List subcommands
var listDeploymentsCmd = &cobra.Command{
	Use:     "deployments",
	Short:   "List Kubernetes deployments",
	Aliases: []string{"deploy", "deployment"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := listDeployments(); err != nil {
			log.Error().Err(err).Msg("Failed to list deployments")
			os.Exit(1)
		}
	},
}

var listPodsCmd = &cobra.Command{
	Use:     "pods",
	Short:   "List Kubernetes pods",
	Aliases: []string{"pod", "po"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := listPods(); err != nil {
			log.Error().Err(err).Msg("Failed to list pods")
			os.Exit(1)
		}
	},
}

// Create subcommands
var createDeploymentCmd = &cobra.Command{
	Use:     "deployment [name] [image]",
	Short:   "Create a Kubernetes deployment",
	Aliases: []string{"deploy"},
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		image := args[1]
		replicas, _ := cmd.Flags().GetInt32("replicas")
		if err := createDeployment(name, image, replicas); err != nil {
			log.Error().Err(err).Msg("Failed to create deployment")
			os.Exit(1)
		}
	},
}

var createPodCmd = &cobra.Command{
	Use:     "pod [name] [image]",
	Short:   "Create a Kubernetes pod",
	Aliases: []string{"po"},
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		image := args[1]
		if err := createPod(name, image); err != nil {
			log.Error().Err(err).Msg("Failed to create pod")
			os.Exit(1)
		}
	},
}

// Delete subcommands
var deleteDeploymentCmd = &cobra.Command{
	Use:     "deployment [name]",
	Short:   "Delete a Kubernetes deployment",
	Aliases: []string{"deploy"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := deleteDeployment(name); err != nil {
			log.Error().Err(err).Msg("Failed to delete deployment")
			os.Exit(1)
		}
	},
}

var deletePodCmd = &cobra.Command{
	Use:     "pod [name]",
	Short:   "Delete a Kubernetes pod",
	Aliases: []string{"po"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := deletePod(name); err != nil {
			log.Error().Err(err).Msg("Failed to delete pod")
			os.Exit(1)
		}
	},
}

// Helper functions
func getKubeClient() (*kubernetes.Clientset, error) {
	kubeconfigPath := getKubeconfigPath()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}
	return kubernetes.NewForConfig(config)
}

func getKubeconfigPath() string {
	if kubeconfig != "" {
		return kubeconfig
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func listDeployments() error {
	log.Info().Str("namespace", namespace).Msg("Listing deployments")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments.Items) == 0 {
		fmt.Printf("No deployments found in namespace '%s'\n", namespace)
		return nil
	}

	fmt.Printf("Found %d deployment(s) in namespace '%s':\n\n", len(deployments.Items), namespace)

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")

	for _, deployment := range deployments.Items {
		ready := fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
		upToDate := fmt.Sprintf("%d", deployment.Status.UpdatedReplicas)
		available := fmt.Sprintf("%d", deployment.Status.AvailableReplicas)

		age := "unknown"
		if !deployment.CreationTimestamp.Time.IsZero() {
			age = formatAge(time.Since(deployment.CreationTimestamp.Time))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			deployment.Name, ready, upToDate, available, age)
	}

	w.Flush()
	return nil
}

func listPods() error {
	log.Info().Str("namespace", namespace).Msg("Listing pods")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		fmt.Printf("No pods found in namespace '%s'\n", namespace)
		return nil
	}

	fmt.Printf("Found %d pod(s) in namespace '%s':\n\n", len(pods.Items), namespace)

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")

	for _, pod := range pods.Items {
		ready := fmt.Sprintf("%d/%d", getPodReadyContainers(pod), len(pod.Spec.Containers))
		status := string(pod.Status.Phase)
		restarts := getPodRestartCount(pod)

		age := "unknown"
		if !pod.CreationTimestamp.Time.IsZero() {
			age = formatAge(time.Since(pod.CreationTimestamp.Time))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			pod.Name, ready, status, restarts, age)
	}

	w.Flush()
	return nil
}

func createDeployment(name, image string, replicas int32) error {
	log.Info().Str("name", name).Str("image", image).Int32("replicas", replicas).Str("namespace", namespace).Msg("Creating deployment")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = clientset.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	fmt.Printf("Deployment '%s' created successfully in namespace '%s'\n", name, namespace)
	return nil
}

func createPod(name, image string) error {
	log.Info().Str("name", name).Str("image", image).Str("namespace", namespace).Msg("Creating pod")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	fmt.Printf("Pod '%s' created successfully in namespace '%s'\n", name, namespace)
	return nil
}

func deleteDeployment(name string) error {
	log.Info().Str("name", name).Str("namespace", namespace).Msg("Deleting deployment")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	err = clientset.AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	fmt.Printf("Deployment '%s' deleted successfully from namespace '%s'\n", name, namespace)
	return nil
}

func deletePod(name string) error {
	log.Info().Str("name", name).Str("namespace", namespace).Msg("Deleting pod")

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	err = clientset.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	fmt.Printf("Pod '%s' deleted successfully from namespace '%s'\n", name, namespace)
	return nil
}

// Utility functions
func getPodReadyContainers(pod corev1.Pod) int {
	ready := 0
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
	}
	return ready
}

func getPodRestartCount(pod corev1.Pod) int32 {
	var restarts int32
	for _, status := range pod.Status.ContainerStatuses {
		restarts += status.RestartCount
	}
	return restarts
}

func formatAge(duration time.Duration) string {
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	} else {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
}

func init() {
	// Add main commands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)

	// Add subcommands to list
	listCmd.AddCommand(listDeploymentsCmd)
	listCmd.AddCommand(listPodsCmd)

	// Add subcommands to create
	createCmd.AddCommand(createDeploymentCmd)
	createCmd.AddCommand(createPodCmd)

	// Add subcommands to delete
	deleteCmd.AddCommand(deleteDeploymentCmd)
	deleteCmd.AddCommand(deletePodCmd)

	// Global flags for all commands
	persistentFlags := []*cobra.Command{listCmd, createCmd, deleteCmd}
	for _, cmd := range persistentFlags {
		cmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to the kubeconfig file (default: $HOME/.kube/config)")
		cmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	}

	// Specific flags for create deployment
	createDeploymentCmd.Flags().Int32P("replicas", "r", 1, "Number of replicas for the deployment")
}
