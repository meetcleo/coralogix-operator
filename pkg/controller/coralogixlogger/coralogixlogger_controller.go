package coralogixlogger

import (
	"context"

	coralogixv1 "github.com/coralogix/coralogix-operator/pkg/apis/coralogix/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_coralogixlogger")

// Add creates a new CoralogixLogger Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCoralogixLogger{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("coralogixlogger-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CoralogixLogger
	err = c.Watch(&source.Kind{Type: &coralogixv1.CoralogixLogger{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner CoralogixLogger
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &coralogixv1.CoralogixLogger{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCoralogixLogger implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCoralogixLogger{}

// ReconcileCoralogixLogger reconciles a CoralogixLogger object
type ReconcileCoralogixLogger struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CoralogixLogger object and makes changes based on the state read
// and what is in the CoralogixLogger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCoralogixLogger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CoralogixLogger")

	// Fetch the CoralogixLogger instance
	instance := &coralogixv1.CoralogixLogger{}
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

	// Check if this ServiceAccount already exists
	serviceAccount := newServiceAccount(instance)
	if err := controllerutil.SetControllerReference(instance, serviceAccount, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
		err = r.client.Create(context.TODO(), serviceAccount)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Skip: ServiceAccount already exists", "ServiceAccount.Namespace", foundServiceAccount.Namespace, "ServiceAccount.Name", foundServiceAccount.Name)
	}

	// Check if this ClusterRole already exists
	clusterRole := newClusterRole(instance)
	if err := controllerutil.SetControllerReference(instance, clusterRole, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	foundClusterRole := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: clusterRole.Name, Namespace: ""}, foundClusterRole)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ClusterRole", "ClusterRole.Namespace", clusterRole.Namespace, "ClusterRole.Name", clusterRole.Name)
		err = r.client.Create(context.TODO(), clusterRole)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Skip: ClusterRole already exists", "ClusterRole.Namespace", foundClusterRole.Namespace, "ClusterRole.Name", foundClusterRole.Name)
	}

	// Check if this ClusterRoleBinding already exists
	clusterRoleBinding := newClusterRoleBinding(instance)
	if err := controllerutil.SetControllerReference(instance, clusterRoleBinding, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: ""}, foundClusterRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ClusterRoleBinding", "ClusterRoleBinding.Namespace", clusterRoleBinding.Namespace, "ClusterRoleBinding.Name", clusterRoleBinding.Name)
		err = r.client.Create(context.TODO(), clusterRoleBinding)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Skip: ClusterRoleBinding already exists", "ClusterRoleBinding.Namespace", foundClusterRoleBinding.Namespace, "ClusterRoleBinding.Name", foundClusterRoleBinding.Name)
	}

	// Check if this DaemonSet already exists
	daemonSet := newDaemonSet(instance)
	if err := controllerutil.SetControllerReference(instance, daemonSet, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	foundDaemonSet := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, foundDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.client.Create(context.TODO(), daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Skip reconcile: DaemonSet already exists", "DaemonSet.Namespace", foundDaemonSet.Namespace, "DaemonSet.Name", foundDaemonSet.Name)
	return reconcile.Result{}, nil
}

// newServiceAccount returns a ServiceAccount with the same namespace as the cr
func newServiceAccount(cr *coralogixv1.CoralogixLogger) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-service-account",
			Namespace: cr.Namespace,
		},
	}
}

// newClusterRole returns a ClusterRole with the same namespace as the cr
func newClusterRole(cr *coralogixv1.CoralogixLogger) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-service-account-role",
			Namespace: cr.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
					"pods",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
		},
	}
}

// newClusterRoleBinding returns a ClusterRoleBinding with the same namespace as the cr
func newClusterRoleBinding(cr *coralogixv1.CoralogixLogger) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-coralogix-service-account-role-binding",
			Namespace: cr.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind: "ClusterRole",
			Name: "fluentd-coralogix-service-account-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: "fluentd-coralogix-service-account",
				Namespace: cr.Namespace,
			},
		},
	}
}

// newDaemonSet returns a DaemonSet with the same name/namespace as the cr
func newDaemonSet(cr *coralogixv1.CoralogixLogger) *appsv1.DaemonSet {
	var uid int64 = 0
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-fluentd-coralogix",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"k8s-app": cr.Name + "-fluentd-coralogix",
				"kubernetes.io/cluster-service": "true",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"k8s-app": cr.Name + "-fluentd-coralogix"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": cr.Name + "-fluentd-coralogix",
						"kubernetes.io/cluster-service": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "fluentd-coralogix",
						Image: "registry.connect.redhat.com/coralogix/coralogix-fluentd:latest",
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: &corev1.SecurityContext{
							RunAsUser: &uid,
						},
						Env: []corev1.EnvVar{
							{
								Name: "CORALOGIX_PRIVATE_KEY",
								Value: cr.Spec.PrivateKey,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("400Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "varlog",
								MountPath: "/var/log",
							},
							{
								Name: "varlibdockercontainers",
								MountPath: "/var/lib/docker/containers",
								ReadOnly: true,
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
					},
					ServiceAccountName: "fluentd-coralogix-service-account",
					Tolerations: []corev1.Toleration{
						{
							Key: "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
	}
}
