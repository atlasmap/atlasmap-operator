package atlasmap

import (
	"context"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"strconv"
	"strings"

	"k8s.io/client-go/rest"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/config"
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

	var routeAction action
	if isOpenShift {
		routeAction = newRouteAction(log.WithValues("type", "create-route"), mgr)
	} else {
		routeAction = newIngressAction(log.WithValues("type", "create-ingress"), mgr)
	}

	return []action{
		newServiceAction(log.WithValues("type", "service"), mgr),
		routeAction,
		newDeploymentAction(log.WithValues("type", "create-deployment"), mgr),
	}
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
		atlasMap.Status.Phase = phase
		if err := action.client.Status().Update(ctx, atlasMap); err != nil {
			action.log.Error(err, "Error updating AtlasMap status", "phase", phase)
		}
	}
}

func atlasMapLabels(atlasMap *v1alpha1.AtlasMap) map[string]string {
	return map[string]string{"app": "atlasmap", "atlasmap.io/name": atlasMap.ObjectMeta.Name}
}

func atlasMapImage(atlasMap *v1alpha1.AtlasMap) string {
	if len(atlasMap.Spec.Version) == 0 {
		return config.DefaultConfiguration.GetAtlasMapImage()
	}
	return util.ImageName(config.DefaultConfiguration.AtlasMapImage, atlasMap.Spec.Version)
}

func atlasMapProbePath(atlasMap *v1alpha1.AtlasMap) (string, error) {
	// Handle differences in Spring Boot actuator health endpoint path
	if atlasMap.Spec.Version != "" {
		versionParts := strings.Split(atlasMap.Spec.Version, ".")
		if len(versionParts) > 1 {
			major, err := strconv.Atoi(versionParts[0])
			if err != nil {
				return "", err
			}

			minor, err := strconv.Atoi(versionParts[1])
			if err != nil {
				return "", err
			}

			if major == 1 && minor < 43 {
				return "/management/health", nil
			}
		}
	}
	return "/actuator/health", nil
}
