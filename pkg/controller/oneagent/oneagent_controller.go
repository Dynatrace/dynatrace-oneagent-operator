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
	dynatraceApiToken = "apiToken"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

var log = logf.Log.WithName("oneagent.controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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

var _ reconcile.Reconciler = &ReconcileOneAgent{}

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
	reqLogger := log.WithValues("Namespace", request.Namespace, "Name", request.Name)
	reqLogger.Info("Reconciling OneAgent")

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

		reqLogger.Info("Updating custom resource", "Cause", "Defaults", "OneAgent.Status", instance.Status)
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
		reqLogger.Info("Updating custom resource", "Cause", "Rollout", "OneAgent.Status", instance.Status)
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	updateCR, err = r.reconcileVersion(reqLogger, instance)
	if err != nil {
		return reconcile.Result{}, err
	} else if updateCR == true {
		reqLogger.Info("Updating custom resource", "Cause", "Version", "OneAgent.Status", instance.Status)
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
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
		reqLogger.Info("Creating new DaemonSet")
		err = r.client.Create(context.TODO(), dsDesired)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		//TODO use `hasSpecChanged`
		if hasSpecChanged(&dsActual.Spec, &instance.Spec) {
			reqLogger.Info("Updating existing DaemonSet")
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
		reqLogger.Error(err, "Failed to get tokens", "Secret", instance.Spec.Tokens)
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
		reqLogger.Error(err, "Failed to get desired version")
		return false, err
	} else if desired != "" && instance.Status.Version != desired {
		reqLogger.Info("New version available", "Previous", instance.Status.Version, "Desired", desired)
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
		reqLogger.Error(err, "Failed to list pods", "listOps", listOps)
		return updateCR, err
	}

	// determine pods to restart
	podsToDelete, instances := getPodsToRestart(podList.Items, dtc, instance)
	if !reflect.DeepEqual(instances, instance.Status.Items) {
		//TODO logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.items": instances}).Info("status changed")
		reqLogger.Info("OneAgent pod instances changed")
		updateCR = true
		instance.Status.Items = instances
	}

	// restart daemonset
	err = r.deletePods(instance, podsToDelete)
	if err != nil {
		//TODO logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to delete pods")
		reqLogger.Error(err, "Failed to update version")
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
	labels := buildLabels(instance.Name)
	podSpec := newPodSpecForCR(instance)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
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

// getPodsToRestart determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func getPodsToRestart(pods []corev1.Pod, dtc dtclient.Client, instance *dynatracev1alpha1.OneAgent) ([]corev1.Pod, map[string]dynatracev1alpha1.OneAgentInstance) {
	var doomedPods []corev1.Pod
	instances := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		//TODO logrus.WithFields(logrus.Fields{"instance": instance.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Debug("processing pod")
		item := dynatracev1alpha1.OneAgentInstance{
			PodName: pod.Name,
		}
		ver, err := dtc.GetVersionForIp(pod.Status.HostIP)
		if err != nil {
			//TODO logrus.WithFields(logrus.Fields{"instance": instance.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "hostIP": pod.Status.HostIP, "warning": err}).Warning("no agent found for host")
			// use last know version if available
			if i, ok := instance.Status.Items[pod.Spec.NodeName]; ok {
				item.Version = i.Version
			}
		} else {
			//TODO logrus.WithFields(logrus.Fields{"instance": instance.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "version": ver}).Debug("")
			item.Version = ver
			if ver != instance.Status.Version {
				//TODO logrus.WithFields(logrus.Fields{"instance": instance.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "actual": ver, "desired": instance.Status.Version}).Info("")
				doomedPods = append(doomedPods, pod)
			}
		}
		instances[pod.Spec.NodeName] = item
	}

	return doomedPods, instances
}

// deletePods deletes a list of pods
//
// Returns an error in the following conditions:
//  - failure on object deletion
//  - timeout on waiting for ready state
func (r *ReconcileOneAgent) deletePods(instance *dynatracev1alpha1.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		//TODO logrus.WithFields(logrus.Fields{"oneagent": instance.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("deleting pod")
		err := r.client.Delete(context.TODO(), &pod)
		if err != nil {
			//TODO logrus.WithFields(logrus.Fields{"oneagent": instance.Name, "pod": pod.Name, "error": err}).Error("failed to delete pod")
			return err
		}

		// wait for pod on node to get "Running" again
		if err := r.waitPodReadyState(instance, pod); err != nil {
			//TODO logrus.WithFields(logrus.Fields{"oneagent": instance.Name, "nodeName": pod.Spec.NodeName, "warning": err}).Warning("timeout waiting on pod to get ready")
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
			//TODO logrus.WithFields(logrus.Fields{"oneagent": instance.Name, "nodeName": pod.Spec.NodeName, "pods": podList, "warning": status}).Warning("failed to query pods")
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
