package atlasmap

import (
	"context"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type installRouteAction struct {
	baseAction
}

func newInstallRouteAction(log logr.Logger, mgr manager.Manager) action {
	return &installRouteAction{
		newBaseAction(log, mgr),
	}
}

func (action *installRouteAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	service := &corev1.Service{}
	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		service = createAtlasMapService(atlasMap)
		err := action.deployResource(ctx, atlasMap, service)
		if err != nil {
			action.log.Error(err, "Error creating Service.", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return err
		}
	} else if err != nil {
		action.log.Error(err, "Error retrieving Service.", "Service.Namespace", atlasMap.Namespace, "Service.Name", atlasMap.Name)
		return err
	}

	route := &routev1.Route{}
	err = action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		route = createAtlasMapRoute(atlasMap)
		err := action.deployResource(ctx, atlasMap, route)

		// Route can take a while to create so there's a chance of an 'already exists' error occurring
		if err != nil && !errors.IsAlreadyExists(err) {
			action.log.Error(err, "Error creating Route.", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
			return err
		}
	} else if err != nil {
		action.log.Error(err, "Error retrieving Route.", "Route.Namespace", atlasMap.Namespace, "Route.Name", atlasMap.Name)
		return err
	}

	return nil
}

func createAtlasMapService(atlasMap *v1alpha1.AtlasMap) *corev1.Service {
	return &corev1.Service{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      atlasMap.ObjectMeta.Name,
			Namespace: atlasMap.ObjectMeta.Namespace,
			Labels:    atlasMapLabels(atlasMap),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: atlasMapLabels(atlasMap),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: portAtlasMap,
				},
			},
		},
	}
}

func createAtlasMapRoute(atlasMap *v1alpha1.AtlasMap) *routev1.Route {
	return &routev1.Route{
		TypeMeta: v1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      atlasMap.Name,
			Namespace: atlasMap.Namespace,
			Labels:    atlasMapLabels(atlasMap),
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(atlasMap, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    atlasMap.Kind,
				}),
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: atlasMap.Name,
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}
}
