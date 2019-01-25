package oneagent

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"

	dynatracev1alpha1 "github.com/dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	dynatracePaasToken = "paasToken"
	dynatraceApiToken  = "apiToken"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

var log = logf.Log.WithName("oneagent.controller")

// Add creates a new OneAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	cfg, _ := config.GetConfig()
	c, _ := client.New(cfg, client.Options{})
	return &ReconcileOneAgent{client: c, scheme: mgr.GetScheme()}
	//return &ReconcileOneAgent{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("oneagent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OneAgent
	err = c.Watch(&source.Kind{Type: &dynatracev1alpha1.OneAgent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSets and requeue the owner OneAgent
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dynatracev1alpha1.OneAgent{},
	})
	if err != nil {
		return err
	}

	return nil
}

// ReconcileOneAgent reconciles a OneAgent object
type ReconcileOneAgent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("namespace", request.Namespace, "name", request.Name)
	reqLogger.Info("reconciling oneagent")

	// Fetch the OneAgent instance
	instance := &dynatracev1alpha1.OneAgent{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not dsActual, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := validate(instance); err != nil {
		reqLogger.WithValues()
		return reconcile.Result{}, err
	}

	// default value for .spec.tokens
	if instance.Spec.Tokens == "" {
		instance.Spec.Tokens = instance.Name

		reqLogger.Info("updating custom resource", "cause", "defaults applied")
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	var updateCR bool

	updateCR, err = r.reconcileRollout(reqLogger, instance)
	if err != nil {
		return reconcile.Result{}, err
	} else if updateCR == true {
		reqLogger.Info("updating custom resource", "cause", "initial rollout")
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	if instance.Spec.DisableAgentUpdate == true {
		reqLogger.Info("automatic oneagent update is disabled")
		return reconcile.Result{}, nil
	} else {
		updateCR, err = r.reconcileVersion(reqLogger, instance)
		if err != nil {
			return reconcile.Result{}, err
		} else if updateCR == true {
			reqLogger.Info("updating custom resource", "cause", "version upgrade", "status", instance.Status)
			err := r.updateCR(instance)
			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
		}
	}

	return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
}

func (r *ReconcileOneAgent) reconcileRollout(reqLogger logr.Logger, instance *dynatracev1alpha1.OneAgent) (bool, error) {
	updateCR := false

	// element needs to be inserted before it is used in ONEAGENT_INSTALLER_SCRIPT_URL
	if instance.Spec.Env[0].Name != "ONEAGENT_INSTALLER_TOKEN" {
		instance.Spec.Env = append(instance.Spec.Env[:0], append([]corev1.EnvVar{{
			Name: "ONEAGENT_INSTALLER_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.Tokens},
					Key:                  dynatracePaasToken}},
		}}, instance.Spec.Env[0:]...)...)
		updateCR = true
	}

	// Define a new DaemonSet object
	dsDesired := newDaemonSetForCR(instance)

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	dsActual := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dsDesired.Name, Namespace: dsDesired.Namespace}, dsActual)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("creating new daemonset")
		err = r.client.Create(context.TODO(), dsDesired)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		if hasSpecChanged(&dsActual.Spec, &instance.Spec) {
			reqLogger.Info("updating existing daemonset")
			err = r.client.Update(context.TODO(), dsDesired)
			if err != nil {
				return false, err
			}
		}
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) reconcileVersion(reqLogger logr.Logger, instance *dynatracev1alpha1.OneAgent) (bool, error) {
	updateCR := false

	secret, err := r.getSecret(instance.Spec.Tokens, instance.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to get tokens", "secret", instance.Spec.Tokens)
		return false, nil
	}

	if err = verifySecret(secret); err != nil {
		return false, err
	}

	// initialize dynatrace client
	var certificateValidation = dtclient.SkipCertificateValidation(instance.Spec.SkipCertCheck)
	apiToken, _ := getToken(secret, dynatraceApiToken)
	paasToken, _ := getToken(secret, dynatracePaasToken)
	dtc, err := dtclient.NewClient(instance.Spec.ApiUrl, apiToken, paasToken, certificateValidation)
	if err != nil {
		return false, err
	}

	// get desired version
	desired, err := dtc.GetVersionForLatest(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		reqLogger.Error(err, "failed to get desired version")
		return false, err
	} else if desired != "" && instance.Status.Version != desired {
		reqLogger.Info("new version available", "actual", instance.Status.Version, "desired", desired)
		instance.Status.Version = desired
		updateCR = true
	}

	// query oneagent pods
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(buildLabels(instance.Name))
	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labelSelector,
	}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "failed to list pods", "listops", listOps)
		return updateCR, err
	}

	// determine pods to restart
	podsToDelete, instances := getPodsToRestart(podList.Items, dtc, instance)
	if !reflect.DeepEqual(instances, instance.Status.Items) {
		reqLogger.Info("oneagent pod instances changed")
		updateCR = true
		instance.Status.Items = instances
	}

	// restart daemonset
	err = r.deletePods(instance, podsToDelete)
	if err != nil {
		reqLogger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) updateCR(instance *dynatracev1alpha1.OneAgent) error {
	instance.Status.UpdatedTimestamp = metav1.Now()

	return r.client.Update(context.TODO(), instance)
}

// getSecret retrieves a secret containing PaaS and API tokens for Dynatrace API.
//
// Returns an error if the secret is not found.
func (r *ReconcileOneAgent) getSecret(name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	err := r.client.Get(context.TODO(), key, secret)
	if err != nil && errors.IsNotFound(err) {
		return &corev1.Secret{}, err
	}

	return secret, nil
}

func newDaemonSetForCR(instance *dynatracev1alpha1.OneAgent) *appsv1.DaemonSet {
	selector := buildLabels(instance.Name)
	podSpec := newPodSpecForCR(instance)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    selector,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selector},
				Spec:       podSpec,
			},
		},
	}
}

func newPodSpecForCR(instance *dynatracev1alpha1.OneAgent) corev1.PodSpec {
	trueVar := true

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            instance.Spec.Args,
			Env:             instance.Spec.Env,
			Image:           instance.Spec.Image,
			ImagePullPolicy: corev1.PullAlways,
			Name:            "dynatrace-oneagent",
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"pgrep", "-f", "oneagentwatchdog"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       30,
			},
			Resources: instance.Spec.Resources,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &trueVar,
			},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "host-root",
				MountPath: "/mnt/root",
			}},
		}},
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            true,
		NodeSelector:       instance.Spec.NodeSelector,
		PriorityClassName:  instance.Spec.PriorityClassName,
		ServiceAccountName: "dynatrace-oneagent",
		Tolerations:        instance.Spec.Tolerations,
		Volumes: []corev1.Volume{{
			Name: "host-root",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		}},
	}
}

// deletePods deletes a list of pods
//
// Returns an error in the following conditions:
//  - failure on object deletion
//  - timeout on waiting for ready state
func (r *ReconcileOneAgent) deletePods(instance *dynatracev1alpha1.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		err := r.client.Delete(context.TODO(), &pod)
		if err != nil {
			return err
		}

		// wait for pod on node to get "Running" again
		if err := r.waitPodReadyState(instance, pod); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileOneAgent) waitPodReadyState(instance *dynatracev1alpha1.OneAgent, pod corev1.Pod) error {
	var status error

	fieldSelector, _ := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%v,status.phase=Running,metadata.name!=%v", pod.Spec.NodeName, pod.Name))
	labelSelector := labels.SelectorFromSet(buildLabels(instance.Name))
	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	for splay := uint16(0); splay < *instance.Spec.WaitReadySeconds; splay += splayTimeSeconds {
		time.Sleep(time.Duration(splayTimeSeconds) * time.Second)
		podList := &corev1.PodList{}
		status = r.client.List(context.TODO(), listOps, podList)
		if status != nil {
			continue
		}
		if n := len(podList.Items); n == 1 && getPodReadyState(&podList.Items[0]) {
			break
		} else if n > 1 {
			status = fmt.Errorf("too many pods found: expected=1 actual=%d", n)
		}
	}
	return status
}
