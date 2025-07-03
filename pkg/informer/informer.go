package informer

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	deploymentInformer cache.SharedIndexInformer
	podInformer        cache.SharedIndexInformer
)

// StartDeploymentInformer starts a shared informer for Deployments in the default namespace.
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset) {
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		30*time.Second,
		informers.WithNamespace("default"),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()
		}),
	)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Info().Msgf("Deployment added: %s", getDeploymentName(obj))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Info().Msgf("Deployment updated: %s", getDeploymentName(newObj))
		},
		DeleteFunc: func(obj interface{}) {
			log.Info().Msgf("Deployment deleted: %s", getDeploymentName(obj))
		},
	})

	log.Info().Msg("Starting deployment informer...")
	factory.Start(ctx.Done())
	for t, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			log.Error().Msgf("Failed to sync informer for %v", t)
			os.Exit(1)
		}
	}
	log.Info().Msg("Deployment informer cache synced. Watching for events...")
	<-ctx.Done() // Block until context is cancelled
}

// StartPodInformer starts a shared informer for Pods in the default namespace.
func StartPodInformer(ctx context.Context, clientset *kubernetes.Clientset) {
	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		30*time.Second,
		informers.WithNamespace("default"),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()
		}),
	)
	podInformer = factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				log.Info().Str("pod", pod.Name).Str("namespace", pod.Namespace).Str("phase", string(pod.Status.Phase)).Msg("Pod added")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if oldPod, ok := oldObj.(*corev1.Pod); ok {
				if newPod, ok := newObj.(*corev1.Pod); ok {
					if oldPod.Status.Phase != newPod.Status.Phase {
						log.Info().Str("pod", newPod.Name).Str("namespace", newPod.Namespace).
							Str("old_phase", string(oldPod.Status.Phase)).
							Str("new_phase", string(newPod.Status.Phase)).
							Msg("Pod phase updated")
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				log.Info().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("Pod deleted")
			}
		},
	})

	log.Info().Msg("Starting pod informer...")
	factory.Start(ctx.Done())
	for t, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			log.Error().Msgf("Failed to sync informer for %v", t)
			os.Exit(1)
		}
	}
	log.Info().Msg("Pod informer cache synced. Watching for events...")
	<-ctx.Done() // Block until context is cancelled
}

// StartBothInformers starts both deployment and pod informers concurrently.
func StartBothInformers(ctx context.Context, clientset *kubernetes.Clientset) {
	// Start deployment informer in a goroutine
	go StartDeploymentInformer(ctx, clientset)

	// Start pod informer in a goroutine
	go StartPodInformer(ctx, clientset)

	// Wait for context cancellation
	<-ctx.Done()
}

// GetDeploymentNames returns a slice of deployment names from the informer's cache.
func GetDeploymentNames() []string {
	var names []string
	if deploymentInformer == nil {
		return names
	}
	for _, obj := range deploymentInformer.GetStore().List() {
		if d, ok := obj.(*appsv1.Deployment); ok {
			names = append(names, d.Name)
		}
	}
	return names
}

// GetPodNames returns a slice of pod names from the informer's cache.
func GetPodNames() []string {
	var names []string
	if podInformer == nil {
		return names
	}
	for _, obj := range podInformer.GetStore().List() {
		if p, ok := obj.(*corev1.Pod); ok {
			names = append(names, p.Name)
		}
	}
	return names
}

func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
