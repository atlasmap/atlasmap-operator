package atlasmap

import (
	"context"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"reflect"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	routev1 "github.com/openshift/api/route/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// var log = logf.Log.WithName("controller_atlasmap")

// Add creates a new AtlasMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	err := consolev1.Install(mgr.GetScheme())
	if err != nil {
		return err
	}

	newReconciler := &ReconcileAtlasMap{
		client: mgr.GetClient(),
		config: mgr.GetConfig(),
		scheme: mgr.GetScheme(),
	}

	configClient, err := configv1client.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	newReconciler.configClient = configClient
	return add(mgr, newReconciler)
}

var actions []action

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("atlasmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AtlasMap
	err = c.Watch(&source.Kind{Type: &v1alpha1.AtlasMap{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployment and requeue the owner AtlasMap
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.AtlasMap{},
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldDeployment := e.ObjectOld.(*appsv1.Deployment)
				newDeployment := e.ObjectNew.(*appsv1.Deployment)
				return !reflect.DeepEqual(oldDeployment.Spec, newDeployment.Spec) ||
					   oldDeployment.Status.ReadyReplicas != newDeployment.Status.ReadyReplicas
			},
		})
	if err != nil {
		return err
	}

	isOpenShift, err := util.IsOpenShift(mgr.GetConfig())
	if err != nil {
		return err
	}

	if isOpenShift {
		// Watch for changes to secondary resource route and requeue the owner AtlasMap
		err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.AtlasMap{},
		})
		if err != nil {
			return err
		}
	} else {
		// Watch for changes to secondary resource ingress and requeue the owner AtlasMap
		err = c.Watch(&source.Kind{Type: &v1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.AtlasMap{},
		})
		if err != nil {
			return err
		}
	}

	actions = newOperatorActions(log, mgr)

	return nil
}

var _ reconcile.Reconciler = &ReconcileAtlasMap{}

// ReconcileAtlasMap reconciles a AtlasMap object
type ReconcileAtlasMap struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
	configClient *configv1client.Clientset
}

// Reconcile reads that state of the cluster for a AtlasMap object and makes changes based on the state read
// and what is in the AtlasMap.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAtlasMap) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AtlasMap")

	// Fetch the AtlasMap instance
	ctx := context.TODO()
	instance := &v1alpha1.AtlasMap{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			instance.ObjectMeta = metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			}

			isOpenShift, _ := util.IsOpenShift(r.config)
			if isOpenShift && util.IsOpenShift43Plus(r.config){
				//Handle removal of cluster-scope object.
				r.removeConsoleLink(instance)
			}

			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, a := range actions {
		log.Info("Running action: " + a.getName())
		if err := a.handle(ctx, instance); err != nil {
			if errors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			reqLogger.Error(err, "Error running action: "+a.getName())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAtlasMap) removeConsoleLink(atlasMap *v1alpha1.AtlasMap) (request reconcile.Result, err error) {
	consoleLinkName := util.ConsoleLinkName(atlasMap)
	consoleLink := &consolev1.ConsoleLink{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	} else {
		err = r.client.Delete(context.TODO(), consoleLink)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, err
}
