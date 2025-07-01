package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestEnvironment wraps the envtest environment
type TestEnvironment struct {
	Env     *envtest.Environment
	Config  *rest.Config
	Client  *kubernetes.Clientset
	Cancel  context.CancelFunc
	TempDir string
}

// SetupTestEnvironment sets up a test Kubernetes environment using envtest
func SetupTestEnvironment() (*TestEnvironment, error) {
	// Create scheme with required types
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add appsv1 to scheme: %w", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add corev1 to scheme: %w", err)
	}

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "k8s-controller-test-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Set up envtest environment
	testEnv := &envtest.Environment{
		Scheme:            scheme,
		CRDDirectoryPaths: []string{
			// Add CRD paths if needed
		},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: "", // Let envtest download binaries automatically
	}

	log.Info().Msg("Starting test Kubernetes API server")
	cfg, err := testEnv.Start()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to start test environment: %w", err)
	}

	// Create Kubernetes client
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		testEnv.Stop()
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Write kubeconfig for external access (useful for debugging)
	kubeconfigPath := "/tmp/envtest.kubeconfig"
	if err := writeKubeconfigFile(cfg, kubeconfigPath); err != nil {
		log.Warn().Err(err).Str("path", kubeconfigPath).Msg("Failed to write kubeconfig file")
	} else {
		log.Info().Str("path", kubeconfigPath).Msg("Kubeconfig written for debugging")
	}

	return &TestEnvironment{
		Env:     testEnv,
		Config:  cfg,
		Client:  client,
		TempDir: tempDir,
	}, nil
}

// Cleanup cleans up the test environment
func (te *TestEnvironment) Cleanup() {
	if te.Cancel != nil {
		te.Cancel()
	}
	if te.Env != nil {
		log.Info().Msg("Stopping test Kubernetes API server")
		te.Env.Stop()
	}
	if te.TempDir != "" {
		os.RemoveAll(te.TempDir)
	}

	// Clean up the debug kubeconfig
	kubeconfigPath := "/tmp/envtest.kubeconfig"
	if _, err := os.Stat(kubeconfigPath); err == nil {
		os.Remove(kubeconfigPath)
	}
}

// CreateTestDeployment creates a test deployment in the given namespace
func (te *TestEnvironment) CreateTestDeployment(name, namespace, image string, replicas int32) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":  name,
				"test": "true",
			},
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

	return te.Client.AppsV1().Deployments(namespace).Create(
		context.Background(),
		deployment,
		metav1.CreateOptions{},
	)
}

// CreateTestNamespace creates a test namespace
func (te *TestEnvironment) CreateTestNamespace(name string) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test": "true",
			},
		},
	}

	return te.Client.CoreV1().Namespaces().Create(
		context.Background(),
		namespace,
		metav1.CreateOptions{},
	)
}

// UpdateTestDeployment updates a test deployment
func (te *TestEnvironment) UpdateTestDeployment(name, namespace string, replicas int32) (*appsv1.Deployment, error) {
	deployment, err := te.Client.AppsV1().Deployments(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	deployment.Spec.Replicas = &replicas

	return te.Client.AppsV1().Deployments(namespace).Update(
		context.Background(),
		deployment,
		metav1.UpdateOptions{},
	)
}

// DeleteTestDeployment deletes a test deployment
func (te *TestEnvironment) DeleteTestDeployment(name, namespace string) error {
	return te.Client.AppsV1().Deployments(namespace).Delete(
		context.Background(),
		name,
		metav1.DeleteOptions{},
	)
}

// WaitForDeployment waits for a deployment to be ready
func (te *TestEnvironment) WaitForDeployment(name, namespace string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s/%s to be ready", namespace, name)
		default:
			deployment, err := te.Client.AppsV1().Deployments(namespace).Get(
				context.Background(),
				name,
				metav1.GetOptions{},
			)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// writeKubeconfigFile writes a kubeconfig file for the test environment
func writeKubeconfigFile(config *rest.Config, path string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: envtest
contexts:
- context:
    cluster: envtest
    user: envtest
  name: envtest
current-context: envtest
users:
- name: envtest
  user:
    token: %s
`, config.Host, config.BearerToken)

	// If we have TLS config, use it instead of insecure
	if config.CAData != nil {
		kubeconfigContent = fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: envtest
contexts:
- context:
    cluster: envtest
    user: envtest
  name: envtest
current-context: envtest
users:
- name: envtest
  user:
    client-certificate-data: %s
    client-key-data: %s
`, config.Host,
			string(config.CAData),
			string(config.CertData),
			string(config.KeyData))
	}

	return os.WriteFile(path, []byte(kubeconfigContent), 0644)
}

// GetConfig returns the rest config for the test environment
func (te *TestEnvironment) GetConfig() *rest.Config {
	return te.Config
}

// GetClient returns the Kubernetes client for the test environment
func (te *TestEnvironment) GetClient() *kubernetes.Clientset {
	return te.Client
}

// GetKubeconfigPath returns the path to the generated kubeconfig file
func (te *TestEnvironment) GetKubeconfigPath() string {
	return "/tmp/envtest.kubeconfig"
}
