package atlasmap

import (
	"context"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type serviceAction struct {
	baseAction
}

func newServiceAction(log logr.Logger, mgr manager.Manager) action {
	return &serviceAction{
		newBaseAction(log, mgr, "Service"),
	}
}

func (action *serviceAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
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
