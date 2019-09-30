package oneagent

import (
	"context"
	"fmt"
	"reflect"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/nodes"
	oneagent_utils "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent-utils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

// Add creates a new OneAgent NodesController and adds it to the Manager. The Manager will set fields on the NodesController
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	reconciler := NewOneAgentReconciler(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(),
		log.Log.WithName("oneagent.controller"), nil)
	reconciler.dynatraceClientFunc = reconciler.buildDynatraceClient

	return reconciler
}

// NewOneAgentReconciler - initialise a new ReconcileOneAgent instance
func NewOneAgentReconciler(client client.Client, scheme *runtime.Scheme, config *rest.Config, logger logr.Logger,
	dynatraceClientFunc DynatraceClientFunc) *ReconcileOneAgent {

	return &ReconcileOneAgent{
		client:              client,
		scheme:              scheme,
		config:              config,
		logger:              logger,
		dynatraceClientFunc: dynatraceClientFunc,
		nodesController:     nodes.NewController(config),
		istioController:     istio.NewController(config),
	}
}

// add adds a new NodesController to mgr with r as the reconcile.Reconciler
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

	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})

	return nil
}

// DynatraceClientFunc defines handler func for dynatrace client
type DynatraceClientFunc func(*dynatracev1alpha1.OneAgent) (dtclient.Client, error)

// ReconcileOneAgent reconciles a OneAgent object
type ReconcileOneAgent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *rest.Config
	logger logr.Logger

	dynatraceClientFunc DynatraceClientFunc
	nodesController     *nodes.Controller
	istioController     *istio.Controller
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The NodesController will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", request.Namespace, "name", request.Name)
	logger.Info("reconciling oneagent")

	if len(request.Namespace) == 0 {
		return reconcile.Result{}, r.nodesController.ReconcileNodes(request.Name)
	}

	instance := &dynatracev1alpha1.OneAgent{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not dsActual, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}
	r.scheme.Default(instance)

	// if err := validate(instance); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// default value for .spec.tokens
	if instance.Spec.Tokens == "" {
		instance.Spec.Tokens = instance.Name

		logger.Info("updating custom resource", "cause", "defaults applied")
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	dtc, err := r.dynatraceClientFunc(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.EnableIstio {
		if upd, ok := r.istioController.ReconcileIstio(instance, dtc); ok && upd {
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	}

	var updateCR bool

	updateCR, err = r.reconcileRollout(logger, instance)
	if err != nil {
		return reconcile.Result{}, err
	} else if updateCR {
		logger.Info("updating custom resource", "cause", "initial rollout")
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	if instance.Spec.DisableAgentUpdate {
		logger.Info("automatic oneagent update is disabled")
		return reconcile.Result{}, nil
	}

	updateCR, err = r.reconcileVersion(logger, instance, dtc)
	if err != nil {
		return reconcile.Result{}, err
	} else if updateCR {
		logger.Info("updating custom resource", "cause", "version upgrade", "status", instance.Status)
		err := r.updateCR(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
}

func (r *ReconcileOneAgent) reconcileRollout(logger logr.Logger, instance *dynatracev1alpha1.OneAgent) (bool, error) {
	updateCR := false

	// element needs to be inserted before it is used in ONEAGENT_INSTALLER_SCRIPT_URL
	if instance.Spec.Env[0].Name != "ONEAGENT_INSTALLER_TOKEN" {
		instance.Spec.Env = append(instance.Spec.Env[:0], append([]corev1.EnvVar{{
			Name: "ONEAGENT_INSTALLER_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.Tokens},
					Key:                  oneagent_utils.DynatracePaasToken}},
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
		logger.Info("creating new daemonset")
		err = r.client.Create(context.TODO(), dsDesired)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		if hasSpecChanged(&dsActual.Spec, &instance.Spec) {
			logger.Info("updating existing daemonset")
			err = r.client.Update(context.TODO(), dsDesired)
			if err != nil {
				return false, err
			}
		}
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) buildDynatraceClient(instance *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {
	secret, err := r.getSecret(instance.Spec.Tokens, instance.Namespace)
	if err != nil {
		return nil, err
	}

	if err = oneagent_utils.VerifySecret(secret); err != nil {
		return nil, err
	}

	// initialize dynatrace client
	var certificateValidation = dtclient.SkipCertificateValidation(instance.Spec.SkipCertCheck)
	apiToken, _ := oneagent_utils.ExtractToken(secret, oneagent_utils.DynatraceApiToken)
	paasToken, _ := oneagent_utils.ExtractToken(secret, oneagent_utils.DynatracePaasToken)
	dtc, err := dtclient.NewClient(instance.Spec.ApiUrl, apiToken, paasToken, certificateValidation)

	return dtc, err
}

func (r *ReconcileOneAgent) reconcileVersion(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	updateCR := false

	// get desired version
	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		logger.Info("failed to get desired version", "error", err.Error())
		return false, nil
	} else if desired != "" && instance.Status.Version != desired {
		logger.Info("new version available", "actual", instance.Status.Version, "desired", desired)
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
		logger.Error(err, "failed to list pods", "listops", listOps)
		return updateCR, err
	}

	// determine pods to restart
	podsToDelete, instances := getPodsToRestart(podList.Items, dtc, instance)

	// Workaround: 'instances' can be null, making DeepEqual() return false when comparing against an map instance.
	// So, compare as long there is data.
	if (len(instances) > 0 || len(instance.Status.Items) > 0) && !reflect.DeepEqual(instances, instance.Status.Items) {
		logger.Info("oneagent pod instances changed")
		updateCR = true
		instance.Status.Items = instances
	}

	// restart daemonset
	err = r.deletePods(logger, instance, podsToDelete)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) updateCR(instance *dynatracev1alpha1.OneAgent) error {
	instance.Status.UpdatedTimestamp = metav1.Now()

	// client.Update() doesn't apply changes to the .status section, only to .spec. This function also replaces
	// the instance given as a parameter with what it's now currently on Kubernetes, including the old .status value.
	//
	// Because of this we make a copy of this field first.

	newStatus := instance.Status

	// Rather than sending the existing value, the .status.items map also gets replaced in-place, so we send a
	// dummy object to avoid modifying it.
	instance.Status = dynatracev1alpha1.OneAgentStatus{}

	if err := r.client.Update(context.TODO(), instance); err != nil {
		return err
	}

	instance.Status = newStatus

	// Now, with this call we do update the Status section to the new value.
	return r.client.Status().Update(context.TODO(), instance)
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
						Command: []string{
							"/bin/sh", "-c", "grep -q oneagentwatchdo /proc/[0-9]*/stat",
						},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       30,
				TimeoutSeconds:      1,
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
func (r *ReconcileOneAgent) deletePods(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		logger.Info("deleting pod", "pod", pod.Name, "node", pod.Spec.NodeName)

		// delete pod
		err := r.client.Delete(context.TODO(), &pod)
		if err != nil {
			return err
		}

		logger.Info("waiting until pod is ready on node", "node", pod.Spec.NodeName)

		// wait for pod on node to get "Running" again
		if err := r.waitPodReadyState(instance, pod); err != nil {
			return err
		}

		logger.Info("pod recreated successfully on node", "node", pod.Spec.NodeName)
	}

	return nil
}

func (r *ReconcileOneAgent) waitPodReadyState(instance *dynatracev1alpha1.OneAgent, pod corev1.Pod) error {
	var status error

	labelSelector := labels.SelectorFromSet(buildLabels(instance.Name))
	listOps := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labelSelector,
	}

	for splay := uint16(0); splay < *instance.Spec.WaitReadySeconds; splay += splayTimeSeconds {
		time.Sleep(time.Duration(splayTimeSeconds) * time.Second)

		// The actual selector we need is,
		// "spec.nodeName=<pod.Spec.NodeName>,status.phase=Running,metadata.name!=<pod.Name>"
		//
		// However, the client falls back to a cached implementation for .List() after the first attempt, which
		// is not able to handle our query so the function fails. Because of this, we're getting all the pods and
		// filtering it ourselves.
		podList := &corev1.PodList{}
		status = r.client.List(context.TODO(), listOps, podList)
		if status != nil {
			continue
		}

		var foundPods []*corev1.Pod
		for i := range podList.Items {
			p := &podList.Items[i]
			if p.Spec.NodeName != pod.Spec.NodeName || p.Status.Phase != corev1.PodRunning ||
				p.ObjectMeta.Name == pod.Name {
				continue
			}
			foundPods = append(foundPods, p)
		}

		if n := len(foundPods); n == 0 {
			status = fmt.Errorf("waiting for pod to be recreated on node: %s", pod.Spec.NodeName)
		} else if n == 1 && getPodReadyState(foundPods[0]) {
			break
		} else if n > 1 {
			status = fmt.Errorf("too many pods found: expected=1 actual=%d", n)
		}
	}

	return status
}
