package atlasmap

import (
	"context"
	"time"

	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"

	atlasmapv1alpha1 "github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// DefaultImageName is the default image that AtlasMap CRs should use
	DefaultImageName  = "docker.io/atlasmap/atlasmap:latest"
	probeEndpointPath = "/management/health"
)

var log = logf.Log.WithName("controller_atlasmap")

// Add creates a new AtlasMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAtlasMap{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("atlasmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AtlasMap
	err = c.Watch(&source.Kind{Type: &atlasmapv1alpha1.AtlasMap{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner AtlasMap
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &atlasmapv1alpha1.AtlasMap{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAtlasMap{}

// ReconcileAtlasMap reconciles a AtlasMap object
type ReconcileAtlasMap struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
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
	instance := &atlasmapv1alpha1.AtlasMap{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		service = createAtlasMapService(instance)
		err := r.deployResource(instance, service)
		if err != nil {
			reqLogger.Error(err, "Error creating Service.", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return reconcile.Result{}, err
		}
	}

	route := &routev1.Route{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		route = createAtlasMapRoute(instance)
		err := r.deployResource(instance, route)
		if err != nil {
			reqLogger.Error(err, "Error creating Route.", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
			return reconcile.Result{}, err
		}

		// TODO: Fix this hack. Route takes some time to create, so sleep until its likely to be available
		time.Sleep(5 * time.Second)
	} else if err != nil {
		reqLogger.Error(err, "Error retrieving Route.", "Route.Namespace", instance.Namespace, "Route.Name", instance.Name)
		return reconcile.Result{}, err
	}

	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment = createAtlasMapDeployment(instance)
		err := configureResources(instance, &deployment.Spec.Template.Spec.Containers[0])
		if err != nil {
			reqLogger.Error(err, "Error configuring Deployment resources.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}

		err = r.deployResource(instance, deployment)
		if err != nil {
			reqLogger.Error(err, "Error creating Deployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Error retrieving Deployment.", "Deployment.Namespace", instance.Namespace, "Deployment.Name", instance.Name)
		return reconcile.Result{}, err
	}

	// Reconcile desired replicas
	replicas := instance.Spec.Replicas
	if *deployment.Spec.Replicas != replicas {
		*deployment.Spec.Replicas = replicas
		err := r.client.Update(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "Error updating Deployment Replicas.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Update CR status URL from route host
	url := "https://" + route.Spec.Host
	if instance.Status.URL != url {
		instance.Status.URL = url
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Error updating AtlasMap status image.", "AtlasMap.Namespace", instance.Namespace, "AtlasMap.Name", instance.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		// Reconcile image name
		container := &deployment.Spec.Template.Spec.Containers[0]

		image := atlasMapImage(instance)
		if container.Image != image {
			container.Image = image
			err := r.client.Update(context.TODO(), deployment)
			if err != nil {
				reqLogger.Error(err, "Error updating Deployment container image.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}

		if instance.Status.Image != container.Image {
			instance.Status.Image = container.Image
			err := r.client.Status().Update(context.TODO(), instance)
			if err != nil {
				reqLogger.Error(err, "Error updating AtlasMap status image.", "AtlasMap.Namespace", instance.Namespace, "AtlasMap.Name", instance.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}

		// Reconcile resources
		updateResources, err := resourceListChanged(instance, container.Resources)
		if err != nil {
			reqLogger.Error(err, "Error updating container resources")
			return reconcile.Result{}, err
		}

		if updateResources {
			configureResources(instance, &deployment.Spec.Template.Spec.Containers[0])
			err = r.client.Update(context.TODO(), deployment)
			if err != nil {
				reqLogger.Error(err, "Error updating Deployment container image.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAtlasMap) deployResource(cr *v1alpha1.AtlasMap, resource runtime.Object) error {
	err := controllerutil.SetControllerReference(cr, resource.(v1.Object), r.scheme)
	if err != nil {
		return err
	}
	return r.client.Create(context.TODO(), resource)
}

func createAtlasMapService(cr *atlasmapv1alpha1.AtlasMap) *corev1.Service {
	return &corev1.Service{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      cr.ObjectMeta.Name,
			Namespace: cr.ObjectMeta.Namespace,
			Labels:    atlasMapLabels(cr),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: atlasMapLabels(cr),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 8585,
				},
			},
		},
	}
}

func createAtlasMapRoute(cr *v1alpha1.AtlasMap) *routev1.Route {
	return &routev1.Route{
		TypeMeta: v1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    atlasMapLabels(cr),
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   atlasmapv1alpha1.SchemeGroupVersion.Group,
					Version: atlasmapv1alpha1.SchemeGroupVersion.Version,
					Kind:    cr.Kind,
				}),
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: cr.Name,
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}
}

func createAtlasMapDeployment(cr *atlasmapv1alpha1.AtlasMap) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    atlasMapLabels(cr),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cr.Spec.Replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: atlasMapLabels(cr),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: atlasMapLabels(cr),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           atlasMapImage(cr),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Name:            "atlasmap",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8585,
								Name:          "http",
							},
							{
								ContainerPort: 8778,
								Name:          "jolokia",
							},
							{
								ContainerPort: 9779,
								Name:          "prometheus",
							},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromString("http"),
									Path:   probeEndpointPath,
								}},
							InitialDelaySeconds: 60,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Scheme: corev1.URISchemeHTTP,
									Port:   intstr.FromString("http"),
									Path:   probeEndpointPath,
								}},
							InitialDelaySeconds: 15,
							FailureThreshold:    5,
						},
					}},
				},
			},
		},
	}
}

func atlasMapLabels(cr *atlasmapv1alpha1.AtlasMap) map[string]string {
	return map[string]string{"app": "atlasmap", "atlasmap.io/name": cr.ObjectMeta.Name}
}

func atlasMapImage(cr *atlasmapv1alpha1.AtlasMap) string {
	if cr.Spec.Image == "" {
		return DefaultImageName
	}
	return cr.Spec.Image
}
