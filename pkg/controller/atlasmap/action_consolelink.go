package atlasmap

import (
	"context"
	routev1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type consoleLinkAction struct {
	baseAction
}

func newConsoleLinkAction(log logr.Logger, mgr manager.Manager) action {
	return &consoleLinkAction{
		newBaseAction(log, mgr, "ConsoleLink"),
	}
}
func (action *consoleLinkAction) getRoute(ctx context.Context, atlasMap *v1alpha1.AtlasMap) (*routev1.Route, error) {
	route := &routev1.Route{}
	err := action.client.Get(ctx, types.NamespacedName{Name: atlasMap.Name, Namespace: atlasMap.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		return route, nil
	} else if err != nil {
		action.log.Error(err, "Error retrieving route.")
		return nil, err
	}
	return route, err
}

func (action *consoleLinkAction) handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	isOpenShift, err := util.IsOpenShift(action.config)
	if err != nil {
		return err
	}

	if isOpenShift {

		route, err := action.getRoute(ctx, atlasMap)
		if err != nil {
			return err
		}

		consoleLinkName := atlasMap.Name + "-" + atlasMap.Namespace
		consoleLink := &consolev1.ConsoleLink{}
		err = action.client.Get(ctx, types.NamespacedName{Name: consoleLinkName}, consoleLink)
		if err != nil && errors.IsNotFound(err) {
				consoleLink = createNamespaceDashboardLink(consoleLinkName, route, atlasMap)
				err = action.client.Create(ctx, consoleLink)
				if err != nil {
					return err
				}

		} else if err == nil && consoleLink != nil {
			if err := reconcileConsoleLink(route, consoleLink, action.client, ctx); err != nil {
				return err
			}
		}

	}

	return nil
}

func reconcileConsoleLink(route *routev1.Route, link *consolev1.ConsoleLink, client client.Client, ctx context.Context) error {
	url := "https://" + route.Spec.Host
	if link.Spec.Href != url {
		link.Spec.Href = url
		if err := client.Update(ctx, link); err != nil {
			return err
		}
	}

	if link.Spec.Text != route.Name {
		link.Spec.Text = route.Name
		if err := client.Update(ctx, link); err != nil {
			return err
		}
	}

	return nil
}

func createNamespaceDashboardLink(name string, route *routev1.Route, atlasMap *v1alpha1.AtlasMap) *consolev1.ConsoleLink {
	consoleLink := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"app": atlasMap.Name},
		},
		Spec: consolev1.ConsoleLinkSpec{
			Location: consolev1.NamespaceDashboard,
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: []string{atlasMap.Namespace},
			},
		},
	}

	setNamespaceDashboardLink(consoleLink, route)

	return consoleLink
}

func setNamespaceDashboardLink(consoleLink *consolev1.ConsoleLink, route *routev1.Route) {
	consoleLink.Spec.Link.Text = route.Name
	consoleLink.Spec.Link.Href = "https://" + route.Spec.Host
}
