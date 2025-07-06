package ctrl

import (
	context "context"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Info().Msgf("Reconciling Deployment: %s/%s", req.Namespace, req.Name)

	// Fetch the deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, deployment)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info().Msgf("Deployment %s/%s was deleted", req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		log.Error().Err(err).Msgf("Failed to fetch Deployment %s/%s", req.Namespace, req.Name)
		return ctrl.Result{}, err
	}

	// Log deployment details
	r.logDeploymentEvent(deployment)

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) logDeploymentEvent(deployment *appsv1.Deployment) {
	name := deployment.Name
	namespace := deployment.Namespace

	// Get replica counts
	desiredReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}

	currentReplicas := deployment.Status.Replicas
	readyReplicas := deployment.Status.ReadyReplicas
	availableReplicas := deployment.Status.AvailableReplicas
	updatedReplicas := deployment.Status.UpdatedReplicas

	// Log basic deployment info
	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Int32("desired_replicas", desiredReplicas).
		Int32("current_replicas", currentReplicas).
		Int32("ready_replicas", readyReplicas).
		Int32("available_replicas", availableReplicas).
		Int32("updated_replicas", updatedReplicas).
		Msg("Deployment status")

	// Check for scaling events
	if currentReplicas != desiredReplicas {
		if currentReplicas < desiredReplicas {
			log.Info().
				Str("namespace", namespace).
				Str("name", name).
				Int32("from_replicas", currentReplicas).
				Int32("to_replicas", desiredReplicas).
				Msg("🔄 Deployment scaling UP detected")
		} else if currentReplicas > desiredReplicas {
			log.Info().
				Str("namespace", namespace).
				Str("name", name).
				Int32("from_replicas", currentReplicas).
				Int32("to_replicas", desiredReplicas).
				Msg("🔄 Deployment scaling DOWN detected")
		}
	}

	// Check deployment conditions
	for _, condition := range deployment.Status.Conditions {
		log.Info().
			Str("namespace", namespace).
			Str("name", name).
			Str("condition_type", string(condition.Type)).
			Str("status", string(condition.Status)).
			Str("reason", condition.Reason).
			Str("message", condition.Message).
			Time("last_transition", condition.LastTransitionTime.Time).
			Msg("📋 Deployment condition")
	}

	// Log resource requests and limits if present
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Resources.Requests != nil || container.Resources.Limits != nil {
				log.Info().
					Str("namespace", namespace).
					Str("name", name).
					Str("container_name", container.Name).
					Interface("requests", container.Resources.Requests).
					Interface("limits", container.Resources.Limits).
					Msg("📊 Container resources")
			}
		}
	}
}

func AddDeploymentController(mgr manager.Manager) error {
	r := &DeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
