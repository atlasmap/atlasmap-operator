package action

import (
	"context"

	"github.com/atlasmap/atlasmap-operator/controllers/util"
	"k8s.io/client-go/rest"

	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logf.Log.WithName("action")

type Action interface {
	Handle(ctx context.Context, atlasMap *v1alpha1.AtlasMap) error
	GetName() string
}

type baseAction struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
	name   string
}

/*
 * Create new operator actions
 */
func NewOperatorActions(log logr.Logger, mgr manager.Manager) []Action {
	isOpenShift, err := util.IsOpenShift(mgr.GetConfig())
	if err != nil {
		log.Error(err, "Failed to determine cluster version. Defaulting to Kubernetes mode.")
	}

	var consoleLinkAction, routeAction Action
	if isOpenShift {
		routeAction = newRouteAction(log.WithValues("type", "create-route"), mgr)

		if util.IsOpenShift43Plus(mgr.GetConfig()) {
			consoleLinkAction = newConsoleLinkAction(log.WithValues("type", "create-consolelink"), mgr)
		}
	} else {
		routeAction = newIngressAction(log.WithValues("type", "create-ingress"), mgr)
	}

	actions := []Action{
		newServiceAction(log.WithValues("type", "service"), mgr),
		routeAction,
		newDeploymentAction(log.WithValues("type", "create-deployment"), mgr),
	}

	if consoleLinkAction != nil {
		actions = append(actions, consoleLinkAction)
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

func (action *baseAction) GetName() string {
	return action.name
}

func (action *baseAction) deployResource(ctx context.Context, atlasMap *v1alpha1.AtlasMap, resource client.Object) error {
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
