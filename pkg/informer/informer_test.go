package informer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/yourusername/k8s-controller-tutorial/pkg/testutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// Set log level for tests
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestStartDeploymentInformer tests the deployment informer event handling
func TestStartDeploymentInformer(t *testing.T) {
	// Set up test environment
	testEnv, err := testutil.SetupTestEnvironment()
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer testEnv.Cleanup()

	// Create test namespace
	namespace := "test-informer"
	_, err = testEnv.CreateTestNamespace(namespace)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Set up informer with test environment config
	config := InformerConfig{
		Kubeconfig: testEnv.GetKubeconfigPath(),
		InCluster:  false,
		Namespace:  namespace,
		ResyncTime: 1 * time.Second,
	}

	informer, err := NewDeploymentInformer(config)
	if err != nil {
		t.Fatalf("Failed to create deployment informer: %v", err)
	}

	// Track events
	var eventsMutex sync.Mutex
	var events []string

	// Add custom event handlers for testing
	_, err = informer.informer.AddEventHandler(&testEventHandler{
		onAdd: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			eventsMutex.Lock()
			events = append(events, fmt.Sprintf("ADD:%s", deployment.Name))
			eventsMutex.Unlock()
		},
		onUpdate: func(oldObj, newObj interface{}) {
			newDeployment := newObj.(*appsv1.Deployment)
			eventsMutex.Lock()
			events = append(events, fmt.Sprintf("UPDATE:%s", newDeployment.Name))
			eventsMutex.Unlock()
		},
		onDelete: func(obj interface{}) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				// Handle DeletedFinalStateUnknown
				tombstone, ok := obj.(interface{ GetObject() interface{} })
				if ok {
					deployment, ok = tombstone.GetObject().(*appsv1.Deployment)
				}
			}
			if deployment != nil {
				eventsMutex.Lock()
				events = append(events, fmt.Sprintf("DELETE:%s", deployment.Name))
				eventsMutex.Unlock()
			}
		},
	})
	if err != nil {
		t.Fatalf("Failed to add event handler: %v", err)
	}

	// Start informer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := informer.StartDeploymentInformer(ctx); err != nil {
			t.Errorf("Informer failed: %v", err)
		}
	}()

	// Wait for informer to sync
	if !informer.WaitForCacheSync(ctx) {
		t.Fatal("Failed to sync informer cache")
	}

	// Test ADD event
	_, err = testEnv.CreateTestDeployment("test-deployment-1", namespace, "nginx:latest", 1)
	if err != nil {
		t.Fatalf("Failed to create test deployment: %v", err)
	}

	// Test UPDATE event
	time.Sleep(100 * time.Millisecond) // Allow informer to process
	_, err = testEnv.UpdateTestDeployment("test-deployment-1", namespace, 2)
	if err != nil {
		t.Fatalf("Failed to update test deployment: %v", err)
	}

	// Test another ADD event
	time.Sleep(100 * time.Millisecond)
	_, err = testEnv.CreateTestDeployment("test-deployment-2", namespace, "busybox:latest", 1)
	if err != nil {
		t.Fatalf("Failed to create second test deployment: %v", err)
	}

	// Test DELETE event
	time.Sleep(100 * time.Millisecond)
	err = testEnv.DeleteTestDeployment("test-deployment-1", namespace)
	if err != nil {
		t.Fatalf("Failed to delete test deployment: %v", err)
	}

	// Wait for events to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify events were captured
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d: %v", len(events), events)
	}

	// Check specific events
	hasAddEvent := false
	hasUpdateEvent := false
	hasDeleteEvent := false

	for _, event := range events {
		switch {
		case contains(event, "ADD:test-deployment"):
			hasAddEvent = true
		case contains(event, "UPDATE:test-deployment"):
			hasUpdateEvent = true
		case contains(event, "DELETE:test-deployment"):
			hasDeleteEvent = true
		}
	}

	if !hasAddEvent {
		t.Error("ADD event not captured")
	}
	if !hasUpdateEvent {
		t.Error("UPDATE event not captured")
	}
	if !hasDeleteEvent {
		t.Error("DELETE event not captured")
	}

	// Test ListDeployments
	deployments, err := informer.ListDeployments()
	if err != nil {
		t.Fatalf("Failed to list deployments: %v", err)
	}

	// Should have one deployment left (test-deployment-2)
	if len(deployments) != 1 {
		t.Errorf("Expected 1 deployment, got %d", len(deployments))
	}
	if len(deployments) > 0 && deployments[0].Name != "test-deployment-2" {
		t.Errorf("Expected deployment name 'test-deployment-2', got '%s'", deployments[0].Name)
	}

	t.Logf("Successfully captured %d events: %v", len(events), events)

	// Sleep for inspection (uncomment for debugging)
	// t.Log("Sleeping for 5 minutes for manual inspection...")
	// t.Logf("Use: kubectl --kubeconfig=%s get all -n %s", testEnv.GetKubeconfigPath(), namespace)
	// time.Sleep(5 * time.Minute)
}

// TestGetDeploymentName tests the getDeploymentName utility function
func TestGetDeploymentName(t *testing.T) {
	tests := []struct {
		name       string
		deployment *appsv1.Deployment
		expected   string
	}{
		{
			name: "valid deployment",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
			},
			expected: "test-deployment",
		},
		{
			name:       "nil deployment",
			deployment: nil,
			expected:   "unknown",
		},
		{
			name: "empty name",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDeploymentName(tt.deployment)
			if result != tt.expected {
				t.Errorf("getDeploymentName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStartDeploymentInformer_CoversFunction tests the convenience function
func TestStartDeploymentInformer_CoversFunction(t *testing.T) {
	// Set up test environment
	testEnv, err := testutil.SetupTestEnvironment()
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer testEnv.Cleanup()

	// Create test namespace
	namespace := "test-function"
	_, err = testEnv.CreateTestNamespace(namespace)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Test the convenience function with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This should start and then timeout, which is expected
	err = StartDeploymentInformer(ctx, testEnv.GetKubeconfigPath(), false, namespace)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("StartDeploymentInformer failed with unexpected error: %v", err)
	}
}

// TestInformerWithInClusterConfig tests error handling for in-cluster config
func TestInformerWithInClusterConfig(t *testing.T) {
	// This should fail since we're not running in a cluster
	config := InformerConfig{
		InCluster:  true,
		Namespace:  "default",
		ResyncTime: 1 * time.Second,
	}

	_, err := NewDeploymentInformer(config)
	if err == nil {
		t.Error("Expected error when using in-cluster config outside cluster, got nil")
	}
}

// TestInformerConfigValidation tests configuration validation
func TestInformerConfigValidation(t *testing.T) {
	config := InformerConfig{
		// Neither kubeconfig nor in-cluster specified
		Namespace:  "default",
		ResyncTime: 1 * time.Second,
	}

	_, err := NewDeploymentInformer(config)
	if err == nil {
		t.Error("Expected error when neither kubeconfig nor in-cluster specified, got nil")
	}

	expectedMessage := "either kubeconfig path or in-cluster mode must be specified"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, err.Error())
	}
}

// TestInformerStop tests the Stop method
func TestInformerStop(t *testing.T) {
	// Set up test environment
	testEnv, err := testutil.SetupTestEnvironment()
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer testEnv.Cleanup()

	config := InformerConfig{
		Kubeconfig: testEnv.GetKubeconfigPath(),
		InCluster:  false,
		Namespace:  "default",
		ResyncTime: 1 * time.Second,
	}

	informer, err := NewDeploymentInformer(config)
	if err != nil {
		t.Fatalf("Failed to create deployment informer: %v", err)
	}

	// Test Stop method
	informer.Stop()

	// Calling Stop multiple times should not panic
	informer.Stop()
}

// Helper types and functions

// testEventHandler implements cache.ResourceEventHandler for testing
type testEventHandler struct {
	onAdd    func(obj interface{})
	onUpdate func(oldObj, newObj interface{})
	onDelete func(obj interface{})
}

func (h *testEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	if h.onAdd != nil {
		h.onAdd(obj)
	}
}

func (h *testEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if h.onUpdate != nil {
		h.onUpdate(oldObj, newObj)
	}
}

func (h *testEventHandler) OnDelete(obj interface{}) {
	if h.onDelete != nil {
		h.onDelete(obj)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
