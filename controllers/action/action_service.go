package action

import (
	"context"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type serviceAction struct {
	baseAction
}

func newServiceAction(log logr.Logger, mgr manager.Manager) Action {
	return &serviceAction{
		newBaseAction(log, mgr, "Service"),
	}
}

func (action *serviceAction) Handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	service := &corev1.Service{}

	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		service = createAtlasMapService(atlasMap)

		if err := action.deployResource(ctx, atlasMap, service); err != nil {
			return err
		}
	} else if err != nil {
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
