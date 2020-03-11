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
}

type baseAction struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
}

func newOperatorActions(log logr.Logger, mgr manager.Manager) []action {
	return []action{
		newInstallRouteAction(log.WithValues("type", "create-route"), mgr),
		newInstallDeploymentAction(log.WithValues("type", "create-deployment"), mgr),
		newUpdateAction(log.WithValues("type", "update"), mgr),
	}
}

func newBaseAction(log logr.Logger, mgr manager.Manager) baseAction {
	return baseAction{
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetConfig(),
	}
}

func (action *baseAction) deployResource(ctx context.Context, atlasMap *v1alpha1.AtlasMap, resource runtime.Object) error {
	if err := controllerutil.SetControllerReference(atlasMap, resource.(v1.Object), action.scheme); err != nil {
		return err
	}
	return action.client.Create(ctx, resource)
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