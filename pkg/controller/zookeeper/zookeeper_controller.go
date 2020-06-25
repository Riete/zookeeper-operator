package zookeeper

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	middlewarev1alpha1 "github.com/riete/zookeeper-operator/pkg/apis/middleware/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

var log = logf.Log.WithName("controller_zookeeper")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Zookeeper Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileZookeeper{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("zookeeper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Zookeeper
	err = c.Watch(&source.Kind{Type: &middlewarev1alpha1.Zookeeper{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Zookeeper
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &middlewarev1alpha1.Zookeeper{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileZookeeper implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileZookeeper{}

// ReconcileZookeeper reconciles a Zookeeper object
type ReconcileZookeeper struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Zookeeper object and makes changes based on the state read
// and what is in the Zookeeper.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileZookeeper) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Zookeeper")

	// Fetch the Zookeeper instance
	instance := &middlewarev1alpha1.Zookeeper{}
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

	if instance.Spec.Servers%2 == 0 {
		return reconcile.Result{}, fmt.Errorf("zookeeper cluster should always have an odd number of servers")
	}

	// Define a new service object
	service := newStatefulSetService(instance)

	// Set ZooKeeper instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if service already exists
	serviceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Define a new statefulset object
	statefulset := newStatefulSet(instance)

	// Set ZooKeeper instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, statefulset, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if statefulset already exists
	statefulsetFound := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: statefulset.Name, Namespace: statefulset.Namespace}, statefulsetFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new statefulset", "StatefulSet.Namespace", statefulset.Namespace, "StatefulSet.Name", statefulset.Name)
		err = r.client.Create(context.TODO(), statefulset)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Zookeeper already exists - don't requeue
	reqLogger.Info("Skip reconcile: Zookeeper already exists", "Zookeeper.Namespace", instance.Namespace, "Zookeeper.Name", instance.Name)
	return reconcile.Result{}, nil
}

func newStatefulSetService(zk *middlewarev1alpha1.Zookeeper) *corev1.Service {
	labels := map[string]string{"app": zk.Name}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zk.Name,
			Namespace: zk.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Type:      corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "client",
					Protocol:   corev1.ProtocolTCP,
					Port:       middlewarev1alpha1.ZooKeeperClientPort,
					TargetPort: intstr.FromInt(int(middlewarev1alpha1.ZooKeeperClientPort)),
				},
				{
					Name:       "leader-election",
					Protocol:   corev1.ProtocolTCP,
					Port:       middlewarev1alpha1.ZooKeeperLeaderElectionPort,
					TargetPort: intstr.FromInt(int(middlewarev1alpha1.ZooKeeperLeaderElectionPort)),
				},
				{
					Name:       "server",
					Protocol:   corev1.ProtocolTCP,
					Port:       middlewarev1alpha1.ZooKeeperServerPort,
					TargetPort: intstr.FromInt(int(middlewarev1alpha1.ZooKeeperServerPort)),
				},
			},
		},
	}
}

func newStatefulSet(zk *middlewarev1alpha1.Zookeeper) *appsv1.StatefulSet {
	var securityContext int64 = 1000
	labels := map[string]string{"app": zk.Name}
	storageSize, _ := resource.ParseQuantity(zk.Spec.StorageSize)
	command := fmt.Sprintf("start-zookeeper --servers=%d --data_dir=/var/lib/zookeeper/data "+
		"--data_log_dir=/var/lib/zookeeper/data/log --conf_dir=/opt/zookeeper/conf --client_port=%d "+
		"--election_port=%d --server_port=%d --tick_time=2000 --init_limit=10 --sync_limit=5 --heap=%s "+
		"--max_client_cnxns=60 --snap_retain_count=3 --purge_interval=12 --max_session_timeout=40000 "+
		"--min_session_timeout=4000 --log_level=INFO", zk.Spec.Servers, middlewarev1alpha1.ZooKeeperClientPort,
		middlewarev1alpha1.ZooKeeperLeaderElectionPort, middlewarev1alpha1.ZooKeeperServerPort, zk.Spec.Heap)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zk.Name,
			Namespace: zk.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &zk.Spec.Servers,
			ServiceName: zk.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   middlewarev1alpha1.ZooKeeperImage,
						Name:    zk.Name,
						Command: []string{"sh", "-c", command},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: middlewarev1alpha1.ZooKeeperClientPort,
								Name:          "client",
								Protocol:      corev1.ProtocolTCP,
							},
							{
								ContainerPort: middlewarev1alpha1.ZooKeeperLeaderElectionPort,
								Name:          "leader-election",
								Protocol:      corev1.ProtocolTCP,
							},
							{
								ContainerPort: middlewarev1alpha1.ZooKeeperServerPort,
								Name:          "server",
								Protocol:      corev1.ProtocolTCP,
							},
						},
						LivenessProbe: &corev1.Probe{
							FailureThreshold:    3,
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							TimeoutSeconds:      5,
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"sh",
										"-c",
										fmt.Sprintf("zookeeper-ready %d", middlewarev1alpha1.ZooKeeperClientPort),
									},
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							FailureThreshold:    3,
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							TimeoutSeconds:      5,
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"sh",
										"-c",
										fmt.Sprintf("zookeeper-ready %d", middlewarev1alpha1.ZooKeeperClientPort),
									},
								},
							},
						},
						Resources: zk.Spec.Resources,
					}},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   &securityContext,
						RunAsUser: &securityContext,
					},
				},
			},
		},
	}

	if zk.Spec.StorageClass != "" {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "data", MountPath: "/var/lib/zookeeper"},
		}
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageSize,
					},
				},
				StorageClassName: &zk.Spec.StorageClass,
			},
		}}
	}

	return sts
}
