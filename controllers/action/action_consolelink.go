package action

import (
	"context"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/controllers/util"
	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type consoleLinkAction struct {
	baseAction
}

func newConsoleLinkAction(log logr.Logger, mgr manager.Manager) Action {
	return &consoleLinkAction{
		newBaseAction(log, mgr, "ConsoleLink"),
	}
}

func (action *consoleLinkAction) Handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error {
	isOpenShift, err := util.IsOpenShift(action.config)
	if err != nil {
		return err
	}

	if isOpenShift {
		route, err := action.getAtlasMapRoute(ctx, atlasMap)
		if err != nil {
			return err
		}

		consoleLinkName := util.ConsoleLinkName(atlasMap)
		consoleLink := &consolev1.ConsoleLink{}
		err = action.client.Get(ctx, types.NamespacedName{Name: consoleLinkName}, consoleLink)
		if err != nil && errors.IsNotFound(err) {
			consoleLink = createNamespaceDashboardLink(consoleLinkName, route, atlasMap)
			if err := action.client.Create(ctx, consoleLink); err != nil {
				return err
			}
		} else if err == nil && consoleLink != nil {
			if atlasMap.DeletionTimestamp != nil {
				if err := action.client.Delete(ctx, consoleLink); err != nil {
					action.log.Error(err, "Error deleting console link.")
				}
			}

			if err := reconcileConsoleLink(ctx, atlasMap, route, consoleLink, action.client); err != nil {
				return err
			}
		}
	}

	return nil
}

func reconcileConsoleLink(ctx context.Context, atlasMap *v1alpha1.AtlasMap, route *routev1.Route, link *consolev1.ConsoleLink, client client.Client) error {
	updateConsoleLink := false
	url := "https://" + route.Spec.Host
	if link.Spec.Href != url {
		link.Spec.Href = url
		updateConsoleLink = true
	}

	linkText := util.ConsoleLinkText(atlasMap)
	if link.Spec.Text != linkText {
		link.Spec.Text = linkText
		updateConsoleLink = true
	}

	if updateConsoleLink {
		if err := client.Update(ctx, link); err != nil {
			return err
		}
	}

	return nil
}

func createNamespaceDashboardLink(name string, route *routev1.Route, atlasMap *v1alpha1.AtlasMap) *consolev1.ConsoleLink {
	return &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: atlasMapLabels(atlasMap),
		},
		Spec: consolev1.ConsoleLinkSpec{
			Link: consolev1.Link{
				Text: util.ConsoleLinkText(atlasMap),
				Href: "https://" + route.Spec.Host,
			},
			Location: consolev1.NamespaceDashboard,
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: []string{atlasMap.Namespace},
			},
		},
	}
}

func (action *consoleLinkAction) getAtlasMapRoute(ctx context.Context, atlasMap *v1alpha1.AtlasMap) (*routev1.Route, error) {
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

func (action *consoleLinkAction) RemoveConsoleLink(atlasMap *v1alpha1.AtlasMap) error {
	consoleLinkName := util.ConsoleLinkName(atlasMap)
	consoleLink := &consolev1.ConsoleLink{}
	err := action.client.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if err := action.client.Delete(context.TODO(), consoleLink); err != nil {
			return err
		}
	}
	return nil
}
