package action

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netv1 "k8s.io/api/networking/v1"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/controllers/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ingressAction struct {
	baseAction
}

func newIngressAction(log logr.Logger, mgr manager.Manager) Action {
	return &ingressAction{
		newBaseAction(log, mgr, "Ingress"),
	}
}

func (action *ingressAction) Handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	ingress := &netv1.Ingress{}

	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, ingress)
	if err != nil && errors.IsNotFound(err) {
		ingress = createIngress(atlasMap)
		if err := action.deployResource(ctx, atlasMap, ingress); err != nil {
			return err
		}
	} else if err == nil && ingress != nil {
		if err := reconcileIngress(ctx, ingress, atlasMap, action.client); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func createIngress(atlasMap *v1alpha1.AtlasMap) *netv1.Ingress {
	return &netv1.Ingress{
		TypeMeta: v1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: netv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      atlasMap.Name,
			Namespace: atlasMap.Namespace,
			Labels:    atlasMapLabels(atlasMap),
		},
		Spec: netv1.IngressSpec{
			DefaultBackend: &netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: atlasMap.Name,
					Port: netv1.ServiceBackendPort{
						Name:   "port",
						Number: portAtlasMap,
					},
				},
			},
			Rules: []netv1.IngressRule{
				{
					Host: util.GetIngressHostNameFor(atlasMap),
				},
			},
		},
	}
}

func reconcileIngress(ctx context.Context, ingress *netv1.Ingress, atlasMap *v1alpha1.AtlasMap, client client.Client) error {
	if len(ingress.Spec.Rules) == 1 {
		host := util.GetIngressHostNameFor(atlasMap)
		if host != ingress.Spec.Rules[0].Host {
			ingress.Spec.Rules[0].Host = host
			if err := client.Update(ctx, ingress); err != nil {
				return err
			}
		}

		url := "http://" + ingress.Spec.Rules[0].Host
		if atlasMap.Status.URL != url {
			atlasMap.Status.URL = url
			if err := client.Status().Update(ctx, atlasMap); err != nil {
				return err
			}
		}
	}
	return nil
}
