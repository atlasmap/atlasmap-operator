package atlasmap

import (
	"context"
	"github.com/Masterminds/semver"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"k8s.io/client-go/rest"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("action")

type action interface {
	handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error
	getName() string
}

type baseAction struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
	name   string
}

func newOperatorActions(log logr.Logger, mgr manager.Manager) []action {
	isOpenShift, err := util.IsOpenShift(mgr.GetConfig())
	if err != nil {
		log.Error(err, "Failed to determine cluster version. Defaulting to Kubernetes mode.")
	}

	var consoleLink action
	if isOpenShift {
		openShiftSemVer := util.GetClusterVersionSemVer(mgr.GetConfig())
		if openShiftSemVer != nil {
			constraint43, _ := semver.NewConstraint(">= 4.3")
			isOpenShift43Plus := constraint43.Check(openShiftSemVer)

			if isOpenShift43Plus {
				consoleLink = newConsoleLinkAction(log.WithValues("type", "create-consolelink"), mgr)
			}
		}

	}

	var routeAction action
	if isOpenShift {
		routeAction = newRouteAction(log.WithValues("type", "create-route"), mgr)
	} else {
		routeAction = newIngressAction(log.WithValues("type", "create-ingress"), mgr)
	}

	 actions := []action{
		newServiceAction(log.WithValues("type", "service"), mgr),
		routeAction,
		newDeploymentAction(log.WithValues("type", "create-deployment"), mgr),
	 }

	if consoleLink != nil {
		actions = append(actions, consoleLink)
	}

	return actions
}

func newBaseAction(log logr.Logger, mgr manager.Manager, name string) baseAction {
	return baseAction{
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetConfig(),
		name,
	}
}

func (action *baseAction) getName() string {
	return action.name
}

func (action *baseAction) deployResource(ctx context.Context, atlasMap *v1alpha1.AtlasMap, resource runtime.Object) error {
	if err := controllerutil.SetControllerReference(atlasMap, resource.(v1.Object), action.scheme); err != nil {
		return err
	}
	return action.client.Create(ctx, resource)
}

func (action *baseAction) updatePhase(ctx context.Context, atlasMap *v1alpha1.AtlasMap, phase v1alpha1.AtlasMapPhase) {
	if atlasMap.Status.Phase != phase {
		action.log.Info("AtlasMap phase change", "from", atlasMap.Status.Phase, "to", phase)
		atlasMap.Status.Phase = phase
		if err := action.client.Status().Update(ctx, atlasMap); err != nil {
			action.log.Error(err, "Error updating AtlasMap status", "phase", phase)
		}
	}
}
