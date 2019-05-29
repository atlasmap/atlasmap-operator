package atlasmap

import (
	"context"

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
	healthEndpointPath = "/management/health"
	portAtlasMap       = 8585
	portJolokia        = 8778
	portPrometheus     = 9779
)

type installDeploymentAction struct {
	baseAction
}

func newInstallDeploymentAction(log logr.Logger, mgr manager.Manager) action {
	return &installDeploymentAction{
		newBaseAction(log, mgr),
	}
}

func (action *installDeploymentAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	deployment := &appsv1.Deployment{}
	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment = createAtlasMapDeployment(atlasMap)

		if err := configureResources(atlasMap, &deployment.Spec.Template.Spec.Containers[0]); err != nil {
			return err
		}

		if err := action.deployResource(ctx, atlasMap, deployment); err != nil {
			return err
		}
	} else if err != nil {
		action.log.Error(err, "Error retrieving Deployment.", "Deployment.Namespace", atlasMap.Namespace, "Deployment.Name", atlasMap.Name)
		return err
	}

	return nil
}

func createAtlasMapDeployment(atlasMap *v1alpha1.AtlasMap) *appsv1.Deployment {
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
									Path:   healthEndpointPath,
								}},
							InitialDelaySeconds: 60,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromString("http"),
									Path:   healthEndpointPath,
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
