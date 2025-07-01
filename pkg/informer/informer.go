package informer

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// InformerConfig holds configuration for the informer
type InformerConfig struct {
	Kubeconfig string
	InCluster  bool
	Namespace  string
	ResyncTime time.Duration
}

// DeploymentInformer wraps the Kubernetes informer for deployments
type DeploymentInformer struct {
	clientset *kubernetes.Clientset
	informer  cache.SharedInformer
	factory   informers.SharedInformerFactory
	stopCh    chan struct{}
	namespace string
}

// NewDeploymentInformer creates a new deployment informer
func NewDeploymentInformer(config InformerConfig) (*DeploymentInformer, error) {
	var kubeConfig *rest.Config
	var err error

	// Determine authentication method
	if config.InCluster {
		log.Info().Msg("Using in-cluster authentication")
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else if config.Kubeconfig != "" {
		log.Info().Str("kubeconfig", config.Kubeconfig).Msg("Using kubeconfig authentication")
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either kubeconfig path or in-cluster mode must be specified")
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Set default resync time if not specified
	if config.ResyncTime == 0 {
		config.ResyncTime = 30 * time.Second
	}

	// Create shared informer factory
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		config.ResyncTime,
		informers.WithNamespace(config.Namespace),
	)

	// Get deployment informer
	deploymentInformer := factory.Apps().V1().Deployments().Informer()

	return &DeploymentInformer{
		clientset: clientset,
		informer:  deploymentInformer,
		factory:   factory,
		stopCh:    make(chan struct{}),
		namespace: config.Namespace,
	}, nil
}

// StartDeploymentInformer starts the deployment informer with event handlers
func (di *DeploymentInformer) StartDeploymentInformer(ctx context.Context) error {
	log.Info().Str("namespace", di.namespace).Msg("Starting deployment informer")

	// Add event handlers
	_, err := di.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			deploymentName := getDeploymentName(deployment)
			log.Info().
				Str("event", "ADD").
				Str("deployment", deploymentName).
				Str("namespace", deployment.Namespace).
				Int32("replicas", *deployment.Spec.Replicas).
				Msg("Deployment added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment := oldObj.(*appsv1.Deployment)
			newDeployment := newObj.(*appsv1.Deployment)
			deploymentName := getDeploymentName(newDeployment)

			// Only log if there are meaningful changes
			if oldDeployment.ResourceVersion != newDeployment.ResourceVersion {
				log.Info().
					Str("event", "UPDATE").
					Str("deployment", deploymentName).
					Str("namespace", newDeployment.Namespace).
					Int32("old_replicas", *oldDeployment.Spec.Replicas).
					Int32("new_replicas", *newDeployment.Spec.Replicas).
					Int32("ready_replicas", newDeployment.Status.ReadyReplicas).
					Str("old_version", oldDeployment.ResourceVersion).
					Str("new_version", newDeployment.ResourceVersion).
					Msg("Deployment updated")
			}
		},
		DeleteFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				// Handle DeletedFinalStateUnknown
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					log.Error().Msg("Error decoding object, invalid type")
					return
				}
				deployment, ok = tombstone.Obj.(*appsv1.Deployment)
				if !ok {
					log.Error().Msg("Error decoding object tombstone, invalid type")
					return
				}
			}
			deploymentName := getDeploymentName(deployment)
			log.Info().
				Str("event", "DELETE").
				Str("deployment", deploymentName).
				Str("namespace", deployment.Namespace).
				Msg("Deployment deleted")
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start the informer
	di.factory.Start(di.stopCh)

	// Wait for cache to sync
	log.Info().Msg("Waiting for informer cache to sync")
	if !cache.WaitForCacheSync(ctx.Done(), di.informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	log.Info().Msg("Deployment informer cache synced successfully")

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().Msg("Deployment informer stopping")
	close(di.stopCh)

	return nil
}

// Stop gracefully stops the informer
func (di *DeploymentInformer) Stop() {
	if di.stopCh != nil {
		close(di.stopCh)
	}
}

// GetInformer returns the underlying cache.SharedInformer
func (di *DeploymentInformer) GetInformer() cache.SharedInformer {
	return di.informer
}

// WaitForCacheSync waits for the informer cache to sync
func (di *DeploymentInformer) WaitForCacheSync(ctx context.Context) bool {
	return cache.WaitForCacheSync(ctx.Done(), di.informer.HasSynced)
}

// ListDeployments returns all deployments from the informer's cache
func (di *DeploymentInformer) ListDeployments() ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment

	for _, obj := range di.informer.GetStore().List() {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// getDeploymentName safely extracts deployment name
func getDeploymentName(deployment *appsv1.Deployment) string {
	if deployment == nil {
		return "unknown"
	}
	return deployment.Name
}

// StartDeploymentInformer is a convenience function to start an informer with default settings
func StartDeploymentInformer(ctx context.Context, kubeconfig string, inCluster bool, namespace string) error {
	config := InformerConfig{
		Kubeconfig: kubeconfig,
		InCluster:  inCluster,
		Namespace:  namespace,
		ResyncTime: 30 * time.Second,
	}

	informer, err := NewDeploymentInformer(config)
	if err != nil {
		return fmt.Errorf("failed to create deployment informer: %w", err)
	}

	return informer.StartDeploymentInformer(ctx)
}
