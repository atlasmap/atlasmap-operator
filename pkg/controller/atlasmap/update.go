package atlasmap

import (
	"context"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	atlasMapVersionAnnotation = "atlasmap.io/atlasmapversion"
)

type updateAction struct {
	baseAction
}

func newUpdateAction(log logr.Logger, mgr manager.Manager) action {
	return &updateAction{
		newBaseAction(log, mgr),
	}
}

func (action *updateAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	isOpenShift, err := util.IsOpenShift(action.config)
	if err != nil {
		return err
	}

	// Reconcile status URL
	if isOpenShift {
		route := &routev1.Route{}
		err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, route)
		if err != nil {
			// Route not created yet so wait for next AtlasMap reconcile
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if atlasMap.Spec.RouteHostName != route.Spec.Host {
			route.Spec.Host = atlasMap.Spec.RouteHostName
			if err := action.client.Update(ctx, route); err != nil {
				return err
			}
		}

		url := "https://" + route.Spec.Host
		if atlasMap.Status.URL != url {
			atlasMap.Status.URL = url
			if err := action.client.Status().Update(ctx, atlasMap); err != nil {
				return err
			}
		}
	} else {
		ingress := &v1beta1.Ingress{}
		err = action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, ingress)
		if err != nil {
			// Ingress not created yet so wait for next AtlasMap reconcile
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if len(ingress.Spec.Rules) == 1 {
			host := util.IngressHostName(atlasMap)
			if host != ingress.Spec.Rules[0].Host {
				ingress.Spec.Rules[0].Host = host
				if err := action.client.Update(ctx, ingress); err != nil {
					return err
				}
			}

			url := "http://" + ingress.Spec.Rules[0].Host
			if atlasMap.Status.URL != url {
				atlasMap.Status.URL = url
				if err := action.client.Status().Update(ctx, atlasMap); err != nil {
					return err
				}
			}
		}
	}

	deployment := &appsv1.Deployment{}
	err = action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, deployment)
	if err != nil {
		action.log.Error(err, "Error retrieving deployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return err
	}

	deployment = deployment.DeepCopy()

	// Reconcile replicas
	if annotations := deployment.GetAnnotations(); annotations != nil && annotations[atlasMapVersionAnnotation] == atlasMap.GetResourceVersion() {
		if replicas := deployment.Spec.Replicas; atlasMap.Spec.Replicas != *replicas {
			atlasMap.Spec.Replicas = *replicas
			if err := action.client.Update(ctx, atlasMap); err != nil {
				return err
			}
		}
	} else {
		if replicas := atlasMap.Spec.Replicas; *deployment.Spec.Replicas != replicas {
			deployment.Annotations[atlasMapVersionAnnotation] = atlasMap.GetResourceVersion()
			deployment.Spec.Replicas = &replicas
			if err = action.client.Update(ctx, deployment); err != nil {
				return err
			}
		}
	}

	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) > 0 {
		// Reconcile image name from the AtlasMap spec version
		container := &containers[0]

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

			if err := action.client.Update(ctx, deployment); err != nil {
				return err
			}
		}

		if atlasMap.Status.Image != container.Image {
			atlasMap.Status.Image = container.Image
			if err := action.client.Status().Update(ctx, atlasMap); err != nil {
				return err
			}
		}

		// Reconcile resources
		updateResources, err := resourceListChanged(atlasMap, container.Resources)
		if err != nil {
			action.log.Error(err, "Error updating container resources")
			return err
		}

		if updateResources {
			configureResources(atlasMap, container)
			if err = action.client.Update(ctx, deployment); err != nil {
				return err
			}
		}
	}

	// Update AtlasMap resource version
	instance := &v1alpha1.AtlasMap{}
	err = action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, instance)
	if err != nil {
		return err
	}

	if annotations := deployment.GetAnnotations(); annotations != nil && annotations[atlasMapVersionAnnotation] != instance.GetResourceVersion() {
		deployment.Annotations[atlasMapVersionAnnotation] = instance.GetResourceVersion()
		if err := action.client.Update(ctx, deployment); err != nil {
			return err
		}
	}

	return nil
}
