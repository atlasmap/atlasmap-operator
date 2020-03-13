package atlasmap

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"github.com/go-logr/logr"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ingressAction struct {
	baseAction
}

func newIngressAction(log logr.Logger, mgr manager.Manager) action {
	return &ingressAction{
		newBaseAction(log, mgr, "Ingress"),
	}
}

func (action *ingressAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	ingress := &v1beta1.Ingress{}

	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, ingress)
	if err != nil && errors.IsNotFound(err) {
		ingress = createIngress(atlasMap)
		if err := action.deployResource(ctx, atlasMap, ingress); err != nil {
			return err
		}
	} else if err == nil && ingress != nil {
		if err := reconcileIngress(ingress, atlasMap, action.client, ctx); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func createIngress(atlasMap *v1alpha1.AtlasMap) *v1beta1.Ingress {
	return &v1beta1.Ingress{
		TypeMeta: v1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      atlasMap.Name,
			Namespace: atlasMap.Namespace,
			Labels:    atlasMapLabels(atlasMap),
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: atlasMap.Name,
				ServicePort: intstr.FromInt(portAtlasMap),
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: util.GetIngressHostNameFor(atlasMap),
				},
			},
		},
	}
}

func reconcileIngress(ingress *v1beta1.Ingress, atlasMap *v1alpha1.AtlasMap, client client.Client, ctx context.Context) error {
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
