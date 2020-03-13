package atlasmap

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	atlasMapVersionAnnotation = "atlasmap.io/atlasmapversion"
	portAtlasMap              = 8585
	portJolokia               = 8778
	portPrometheus            = 9779
)

type deploymentAction struct {
	baseAction
}

func newDeploymentAction(log logr.Logger, mgr manager.Manager) action {
	return &deploymentAction{
		newBaseAction(log, mgr, "Deployment"),
	}
}

func (action *deploymentAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	deployment, err := getAtlasMapDeployment(action, ctx, atlasMap)

	if err != nil && errors.IsNotFound(err) {
		probePath, err := atlasMapProbePath(atlasMap)
		if err != nil {
			return err
		}

		deployment = createAtlasMapDeployment(atlasMap, probePath)

		if err := configureResources(atlasMap, &deployment.Spec.Template.Spec.Containers[0]); err != nil {
			return err
		}

		if err := action.deployResource(ctx, atlasMap, deployment); err != nil {
			return err
		}
	} else if err == nil && deployment != nil {
		deployment = deployment.DeepCopy()

		// Reconcile replicas
		if err := reconcileReplicas(deployment, atlasMap, ctx, action); err != nil {
			return err
		}

		containers := deployment.Spec.Template.Spec.Containers
		if len(containers) > 0 {
			// Reconcile AtlasMap image
			if err := reconcileImage(deployment, atlasMap, ctx, action.client); err != nil {
				return err
			}

			// Reconcile resources
			if err := reconcileResources(deployment, atlasMap, ctx, action.client); err != nil {
				return err
			}
		}

		// Update resource version
		if err := updateResourceVersion(deployment, atlasMap, action.client, ctx); err != nil {
			return err
		}
	} else {
		action.log.Error(err, "Error retrieving Deployment.", "Deployment.Namespace", atlasMap.Namespace, "Deployment.Name", atlasMap.Name)
		return err
	}

	return nil
}

func getAtlasMapDeployment(action *deploymentAction, ctx context.Context, atlasMap *v1alpha1.AtlasMap) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, deployment)
	return deployment, err
}

func createAtlasMapDeployment(atlasMap *v1alpha1.AtlasMap, probePath string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        atlasMap.Name,
			Namespace:   atlasMap.Namespace,
			Labels:      atlasMapLabels(atlasMap),
			Annotations: map[string]string{atlasMapVersionAnnotation: atlasMap.GetResourceVersion()},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &atlasMap.Spec.Replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: atlasMapLabels(atlasMap),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: atlasMapLabels(atlasMap),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           atlasMapImage(atlasMap),
						ImagePullPolicy: corev1.PullAlways,
						Name:            "atlasmap",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: portAtlasMap,
								Name:          "http",
							},
							{
								ContainerPort: portJolokia,
								Name:          "jolokia",
							},
							{
								ContainerPort: portPrometheus,
								Name:          "prometheus",
							},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromString("http"),
									Path:   probePath,
								}},
							InitialDelaySeconds: 60,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromString("http"),
									Path:   probePath,
								}},
							InitialDelaySeconds: 15,
							FailureThreshold:    5,
						},
					}},
				},
			},
		},
	}
}

func reconcileReplicas(deployment *appsv1.Deployment, atlasMap *v1alpha1.AtlasMap, ctx context.Context, action *deploymentAction) error {
	// Reconcile Deployment.Spec.Replicas replicas to AtlasMap.Spec.Replicas
	if annotations := deployment.GetAnnotations(); annotations != nil && annotations[atlasMapVersionAnnotation] == atlasMap.GetResourceVersion() {
		if replicas := deployment.Spec.Replicas; atlasMap.Spec.Replicas != *replicas {
			atlasMap.Spec.Replicas = *replicas
			action.updatePhase(ctx, atlasMap, v1alpha1.AtlasMapPhasePhaseInitializing)
			if err := action.client.Update(ctx, atlasMap); err != nil {
				return err
			}
		}
	}

	// Reconcile AtlasMap.Spec.Replicas to Deployment.Spec.Replicas
	if replicas := atlasMap.Spec.Replicas; *deployment.Spec.Replicas != replicas {
		deployment.Annotations[atlasMapVersionAnnotation] = atlasMap.GetResourceVersion()
		deployment.Spec.Replicas = &replicas
		action.updatePhase(ctx, atlasMap, v1alpha1.AtlasMapPhasePhaseInitializing)
		if err := action.client.Update(ctx, deployment); err != nil {
			return err
		}
	}

	if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
		action.updatePhase(ctx, atlasMap, v1alpha1.AtlasMapPhasePhaseInitializing)
	} else {
		action.updatePhase(ctx, atlasMap, v1alpha1.AtlasMapPhasePhaseDeployed)
	}

	return nil
}

func reconcileImage(deployment *appsv1.Deployment, atlasMap *v1alpha1.AtlasMap, ctx context.Context, client client.Client) error {
	container := &deployment.Spec.Template.Spec.Containers[0]
	image := atlasMapImage(atlasMap)

	if container.Image != image {
		container.Image = image

		// Reconcile the endpoint path for health & liveness probes
		probePath, err := atlasMapProbePath(atlasMap)
		if err != nil {
			return err
		}

		if container.LivenessProbe.HTTPGet.Path != probePath {
			container.LivenessProbe.HTTPGet.Path = probePath
		}

		if container.ReadinessProbe.HTTPGet.Path != probePath {
			container.ReadinessProbe.HTTPGet.Path = probePath
		}

		if err := client.Update(ctx, deployment); err != nil {
			return err
		}
	}

	if atlasMap.Status.Image != container.Image {
		atlasMap.Status.Image = container.Image
		if err := client.Status().Update(ctx, atlasMap); err != nil {
			return err
		}
	}
	return nil
}

func reconcileResources(deployment *appsv1.Deployment, atlasMap *v1alpha1.AtlasMap, ctx context.Context, client client.Client) error {
	container := &deployment.Spec.Template.Spec.Containers[0]
	updateResources, err := resourceListChanged(atlasMap, container.Resources)
	if err != nil {
		return err
	}

	if updateResources {
		if err := configureResources(atlasMap, container); err != nil {
			return err
		}
		if err := client.Update(ctx, deployment); err != nil {
			return err
		}
	}

	return nil
}

func updateResourceVersion(deployment *appsv1.Deployment, atlasMap *v1alpha1.AtlasMap, client client.Client, ctx context.Context) error {
	instance := &v1alpha1.AtlasMap{}

	err := client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, instance)
	if err != nil {
		return err
	}

	if annotations := deployment.GetAnnotations(); annotations != nil && annotations[atlasMapVersionAnnotation] != instance.GetResourceVersion() {
		deployment.Annotations[atlasMapVersionAnnotation] = instance.GetResourceVersion()
		if err := client.Update(ctx, deployment); err != nil {
			return err
		}
	}
	return nil
}
