package action

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type routeAction struct {
	baseAction
}

func newRouteAction(log logr.Logger, mgr manager.Manager) Action {
	return &routeAction{
		newBaseAction(log, mgr, "Route"),
	}
}

func (action *routeAction) Handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	route := &routev1.Route{}

	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		route = createAtlasMapRoute(atlasMap)
		err := action.deployResource(ctx, atlasMap, route)

		// Route can take a while to create so there's a chance of an 'already exists' error occurring
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	} else if err == nil && route != nil {
		if err := reconcileRoute(ctx, atlasMap, route, action.client); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func reconcileRoute(ctx context.Context, atlasMap *v1alpha1.AtlasMap, route *routev1.Route, client client.Client) error {
	if atlasMap.Spec.RouteHostName != route.Spec.Host {
		route.Spec.Host = atlasMap.Spec.RouteHostName
		if err := client.Update(ctx, route); err != nil {
			return err
		}
	}

	url := "https://" + route.Spec.Host
	if atlasMap.Status.URL != url {
		atlasMap.Status.URL = url
		if err := client.Status().Update(ctx, atlasMap); err != nil {
			return err
		}
	}
	return nil
}

func createAtlasMapRoute(atlasMap *v1alpha1.AtlasMap) *routev1.Route {
	return &routev1.Route{
		TypeMeta: v1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      atlasMap.Name,
			Namespace: atlasMap.Namespace,
			Labels:    atlasMapLabels(atlasMap),
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(atlasMap, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeBuilder.GroupVersion.Group,
					Version: v1alpha1.SchemeBuilder.GroupVersion.Version,
					Kind:    atlasMap.Kind,
				}),
			},
		},
		Spec: routev1.RouteSpec{
			Host: atlasMap.Spec.RouteHostName,
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
