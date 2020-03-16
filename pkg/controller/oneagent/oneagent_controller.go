package oneagent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

// Add creates a new OneAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return NewOneAgentReconciler(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), mgr.GetConfig(),
		log.Log.WithName("oneagent.controller"), utils.BuildDynatraceClient)
}

// NewOneAgentReconciler - initialise a new ReconcileOneAgent instance
func NewOneAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, logger logr.Logger,
	dynatraceClientFunc utils.DynatraceClientFunc) *ReconcileOneAgent {

	return &ReconcileOneAgent{
		client:              client,
		apiReader:           apiReader,
		scheme:              scheme,
		config:              config,
		logger:              logger,
		dynatraceClientFunc: dynatraceClientFunc,
		istioController:     istio.NewController(config),
	}
}

// add adds a new OneAgentController to mgr with r as the reconcile.Reconciler
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
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	config    *rest.Config
	logger    logr.Logger

	dynatraceClientFunc utils.DynatraceClientFunc
	istioController     *istio.Controller
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", request.Namespace, "name", request.Name)
	logger.Info("reconciling oneagent")

	instance := &dynatracev1alpha1.OneAgent{}
	// Using the apiReader, which does not use caching to prevent a possible race condition where an old version of
	// the OneAgent object is returned from the cache, but it has already been modified on the cluster side
	err := r.apiReader.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not dsActual, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if err := validate(instance); err != nil {
		updateCR := instance.SetPhaseOnError(err)
		if updateCR {
			if errClient := r.updateCR(instance); errClient != nil {
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", err, errClient)
				}
				return reconcile.Result{}, fmt.Errorf("failed to update CR: %w", err)
			}
		}
		return reconcile.Result{}, err
	}

	var updateCR bool

	dtc, updateCR, err := reconcileDynatraceClient(instance, r.client, r.dynatraceClientFunc, metav1.Now())
	if instance.SetPhaseOnError(err) || updateCR {
		if errClient := r.updateCR(instance); errClient != nil {
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", err, errClient)
			}
			return reconcile.Result{}, fmt.Errorf("failed to update CR: %w", err)
		}
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.EnableIstio {
		upd, ok, err := r.istioController.ReconcileIstio(instance, dtc)
		if ok && upd && err != nil {
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	}

	updateCR, err = r.reconcileRollout(logger, instance, dtc)
	if instance.SetPhaseOnError(err) || updateCR {
		logger.Info("updating custom resource", "cause", "initial rollout")
		errClient := r.updateCR(instance)
		if errClient != nil {
			return reconcile.Result{}, errClient
		}
		if err != nil {
			var serr dtclient.ServerError
			if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
				logger.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
				return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
			}
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	} else if err != nil {
		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			logger.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}
		reconcile.Result{}, err
	}

	if instance.Spec.DisableAgentUpdate {
		logger.Info("automatic oneagent update is disabled")
		return reconcile.Result{}, nil
	}

	updateCR, err = r.reconcileVersion(logger, instance, dtc)
	if instance.SetPhaseOnError(err) || updateCR {
		logger.Info("updating custom resource", "cause", "version change")
		errClient := r.updateCR(instance)
		if err != nil || errClient != nil {
			return reconcile.Result{}, errClient
		}
		if err != nil {
			var serr dtclient.ServerError
			if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
				logger.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
				return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
			}
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	} else if err != nil {
		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			logger.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}
		return reconcile.Result{}, err
	}

	// finally we have to determine the correct non error phase
	updateCR, err = r.determineOneAgentPhase(instance)
	if updateCR {
		logger.Info("updating custom resource", "cause", "phase change")
		if errClient := r.updateCR(instance); errClient != nil {
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", err, errClient)
			}
			return reconcile.Result{}, fmt.Errorf("failed to update CR: %w", err)
		}
	}

	return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
}

func (r *ReconcileOneAgent) reconcileRollout(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired := newDaemonSetForCR(instance)

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	dsActual := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dsDesired.Name, Namespace: dsDesired.Namespace}, dsActual)
	if err != nil && k8serrors.IsNotFound(err) {
		logger.Info("creating new daemonset")
		if err = r.client.Create(context.TODO(), dsDesired); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		if hasSpecChanged(&dsActual.Spec, &dsDesired.Spec) {
			logger.Info("updating existing daemonset")
			if err = r.client.Update(context.TODO(), dsDesired); err != nil {
				return false, err
			}
		}
	}

	if instance.Status.Version == "" {
		desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			return updateCR, fmt.Errorf("failed to get desired version: %w", err)
		}

		instance.Status.Version = desired
		instance.SetPhase(dynatracev1alpha1.Deploying)
		updateCR = true
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) determineOneAgentPhase(instance *dynatracev1alpha1.OneAgent) (bool, error) {
	var phaseChanged bool
	dsActual := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, dsActual)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Error
		instance.Status.Phase = dynatracev1alpha1.Error
		return phaseChanged, err
	}

	if dsActual.Status.NumberReady == dsActual.Status.CurrentNumberScheduled {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Running
		instance.Status.Phase = dynatracev1alpha1.Running
	} else {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Deploying
		instance.Status.Phase = dynatracev1alpha1.Deploying
	}

	return phaseChanged, nil
}

func (r *ReconcileOneAgent) reconcileVersion(logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	updateCR := false

	// get desired version
	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return false, fmt.Errorf("failed to get desired version: %w", err)
	} else if desired != "" && instance.Status.Version != desired {
		logger.Info("new version available", "actual", instance.Status.Version, "desired", desired)
		instance.Status.Version = desired
		updateCR = true
	}

	// query oneagent pods
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(buildLabels(instance.Name)),
	}
	err = r.client.List(context.TODO(), podList, listOps...)
	if err != nil {
		logger.Error(err, "failed to list pods", "listops", listOps)
		return updateCR, err
	}

	// determine pods to restart
	podsToDelete, instances, err := getPodsToRestart(podList.Items, dtc, instance)
	if err != nil {
		return updateCR, err
	}

	// Workaround: 'instances' can be null, making DeepEqual() return false when comparing against an empty map instance.
	// So, compare as long there is data.
	if (len(instances) > 0 || len(instance.Status.Instances) > 0) && !reflect.DeepEqual(instances, instance.Status.Instances) {
		logger.Info("oneagent pod instances changed", "status", instance.Status)
		updateCR = true
		instance.Status.Instances = instances
	}

	var waitSecs uint16 = 300
	if instance.Spec.WaitReadySeconds != nil {
		waitSecs = *instance.Spec.WaitReadySeconds
	}

	// restart daemonset
	err = r.deletePods(logger, podsToDelete, buildLabels(instance.Name), waitSecs)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) updateCR(instance *dynatracev1alpha1.OneAgent) error {
	instance.Status.UpdatedTimestamp = metav1.Now()
	return r.client.Status().Update(context.TODO(), instance)
}

func newDaemonSetForCR(instance *dynatracev1alpha1.OneAgent) *appsv1.DaemonSet {
	podSpec := newPodSpecForCR(instance)
	selectorLabels := buildLabels(instance.Name)
	mergedLabels := mergeLabels(instance.Spec.Labels, selectorLabels)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    mergedLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: mergedLabels},
				Spec:       podSpec,
			},
		},
	}
}

func newPodSpecForCR(instance *dynatracev1alpha1.OneAgent) corev1.PodSpec {
	trueVar := true

	img := "docker.io/dynatrace/oneagent:latest"
	if instance.Spec.Image != "" {
		img = instance.Spec.Image
	}

	sa := "dynatrace-oneagent"
	if instance.Spec.ServiceAccountName != "" {
		sa = instance.Spec.ServiceAccountName
	}

	args := instance.Spec.Args
	if instance.Spec.Proxy != nil && (instance.Spec.Proxy.ValueFrom != "" || instance.Spec.Proxy.Value != "") {
		args = append(instance.Spec.Args, "--set-proxy=$(https_proxy)")
	}

	// K8s 1.18+ is expected to drop the "beta.kubernetes.io" labels in favor of "kubernetes.io" which was added on K8s 1.14.
	// To support both older and newer K8s versions we use node affinity.

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            args,
			Env:             prepareEnvVars(instance),
			Image:           img,
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
			VolumeMounts: prepareVolumeMounts(instance),
		}},
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            true,
		NodeSelector:       instance.Spec.NodeSelector,
		PriorityClassName:  instance.Spec.PriorityClassName,
		ServiceAccountName: sa,
		Tolerations:        instance.Spec.Tolerations,
		DNSPolicy:          instance.Spec.DNSPolicy,
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								corev1.NodeSelectorRequirement{
									Key:      "beta.kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								corev1.NodeSelectorRequirement{
									Key:      "beta.kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
						corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								corev1.NodeSelectorRequirement{
									Key:      "kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								corev1.NodeSelectorRequirement{
									Key:      "kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
					},
				},
			},
		},
		Volumes: prepareVolumes(instance),
	}
}

func prepareVolumes(instance *dynatracev1alpha1.OneAgent) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "host-root",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
	}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "certs",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: "certs.pem",
						},
					},
				},
			},
		})
	}

	return volumes
}

func prepareVolumeMounts(instance *dynatracev1alpha1.OneAgent) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "host-root",
			MountPath: "/mnt/root",
		},
	}

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "certs",
			MountPath: "/mnt/dynatrace/certs",
		})
	}

	return volumeMounts
}

func prepareEnvVars(instance *dynatracev1alpha1.OneAgent) []corev1.EnvVar {
	var token, installerURL, skipCert, proxy *corev1.EnvVar

	reserved := map[string]**corev1.EnvVar{
		"ONEAGENT_INSTALLER_TOKEN":           &token,
		"ONEAGENT_INSTALLER_SCRIPT_URL":      &installerURL,
		"ONEAGENT_INSTALLER_SKIP_CERT_CHECK": &skipCert,
		"https_proxy":                        &proxy,
	}

	var envVars []corev1.EnvVar

	for i := range instance.Spec.Env {
		if p := reserved[instance.Spec.Env[i].Name]; p != nil {
			*p = &instance.Spec.Env[i]
			continue
		}
		envVars = append(envVars, instance.Spec.Env[i])
	}

	if token == nil {
		token = &corev1.EnvVar{
			Name: "ONEAGENT_INSTALLER_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: utils.GetTokensName(instance)},
					Key:                  utils.DynatracePaasToken,
				},
			},
		}
	}

	if installerURL == nil {
		installerURL = &corev1.EnvVar{
			Name:  "ONEAGENT_INSTALLER_SCRIPT_URL",
			Value: fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=$(ONEAGENT_INSTALLER_TOKEN)&arch=x86&flavor=default", instance.Spec.ApiUrl),
		}
	}

	if skipCert == nil {
		skipCert = &corev1.EnvVar{
			Name:  "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
			Value: strconv.FormatBool(instance.Spec.SkipCertCheck),
		}
	}

	env := []corev1.EnvVar{*token, *installerURL, *skipCert}

	if proxy == nil {
		if instance.Spec.Proxy != nil {
			if instance.Spec.Proxy.ValueFrom != "" {
				env = append(env, corev1.EnvVar{
					Name: "https_proxy",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.Proxy.ValueFrom},
							Key:                  "proxy",
						},
					},
				})
			} else if instance.Spec.Proxy.Value != "" {
				env = append(env, corev1.EnvVar{
					Name:  "https_proxy",
					Value: instance.Spec.Proxy.Value,
				})
			}
		}
	} else {
		env = append(env, *proxy)
	}

	return append(env, envVars...)
}

// deletePods deletes a list of pods
//
// Returns an error in the following conditions:
//  - failure on object deletion
//  - timeout on waiting for ready state
func (r *ReconcileOneAgent) deletePods(logger logr.Logger, pods []corev1.Pod, labels map[string]string, waitSecs uint16) error {
	for _, pod := range pods {
		logger.Info("deleting pod", "pod", pod.Name, "node", pod.Spec.NodeName)

		// delete pod
		err := r.client.Delete(context.TODO(), &pod)
		if err != nil {
			return err
		}

		logger.Info("waiting until pod is ready on node", "node", pod.Spec.NodeName)

		// wait for pod on node to get "Running" again
		if err := r.waitPodReadyState(pod, labels, waitSecs); err != nil {
			return err
		}

		logger.Info("pod recreated successfully on node", "node", pod.Spec.NodeName)
	}

	return nil
}

func (r *ReconcileOneAgent) waitPodReadyState(pod corev1.Pod, labels map[string]string, waitSecs uint16) error {
	var status error

	listOps := []client.ListOption{
		client.InNamespace(pod.Namespace),
		client.MatchingLabels(labels),
	}

	for splay := uint16(0); splay < waitSecs; splay += splayTimeSeconds {
		time.Sleep(time.Duration(splayTimeSeconds) * time.Second)

		// The actual selector we need is,
		// "spec.nodeName=<pod.Spec.NodeName>,status.phase=Running,metadata.name!=<pod.Name>"
		//
		// However, the client falls back to a cached implementation for .List() after the first attempt, which
		// is not able to handle our query so the function fails. Because of this, we're getting all the pods and
		// filtering it ourselves.
		podList := &corev1.PodList{}
		status = r.client.List(context.TODO(), podList, listOps...)
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

func reconcileDynatraceClient(oa *dynatracev1alpha1.OneAgent, c client.Client, dtcFunc utils.DynatraceClientFunc, now metav1.Time) (dtclient.Client, bool, error) {
	secretName := utils.GetTokensName(oa)

	tokens := []*struct {
		Type              dynatracev1alpha1.OneAgentConditionType
		Key, Value, Scope string
		Timestamp         **metav1.Time
	}{
		{
			Type:      dynatracev1alpha1.PaaSTokenConditionType,
			Key:       utils.DynatracePaasToken,
			Scope:     dtclient.TokenScopeInstallerDownload,
			Timestamp: &oa.Status.LastPaaSTokenProbeTimestamp,
		},
		{
			Type:      dynatracev1alpha1.APITokenConditionType,
			Key:       utils.DynatraceApiToken,
			Scope:     dtclient.TokenScopeDataExport,
			Timestamp: &oa.Status.LastAPITokenProbeTimestamp,
		},
	}

	updateCR := false
	secretKey := oa.Namespace + ":" + secretName
	secret := &corev1.Secret{}
	err := c.Get(context.TODO(), client.ObjectKey{Namespace: oa.Namespace, Name: secretName}, secret)
	if k8serrors.IsNotFound(err) {
		message := fmt.Sprintf("Secret '%s' not found", secretKey)
		updateCR = oa.SetFailureCondition(dynatracev1alpha1.APITokenConditionType, dynatracev1alpha1.ReasonTokenSecretNotFound, message) || updateCR
		updateCR = oa.SetFailureCondition(dynatracev1alpha1.PaaSTokenConditionType, dynatracev1alpha1.ReasonTokenSecretNotFound, message) || updateCR
		return nil, updateCR, fmt.Errorf(message)
	}

	if err != nil {
		return nil, updateCR, err
	}

	valid := true

	for _, t := range tokens {
		v := secret.Data[t.Key]
		if len(v) == 0 {
			updateCR = oa.SetFailureCondition(t.Type, dynatracev1alpha1.ReasonTokenMissing, fmt.Sprintf("Token %s on secret %s missing", t.Key, secretKey)) || updateCR
			valid = false
		}
		t.Value = string(v)
	}

	if !valid {
		return nil, updateCR, fmt.Errorf("issues found with tokens, see status")
	}

	dtc, err := dtcFunc(c, oa)
	if err != nil {
		return nil, updateCR, err
	}

	for _, t := range tokens {
		if strings.TrimSpace(t.Value) != t.Value {
			updateCR = oa.SetFailureCondition(t.Type, dynatracev1alpha1.ReasonTokenUnauthorized,
				fmt.Sprintf("Token on secret %s has leading and/or trailing spaces", secretKey)) || updateCR
			continue
		}

		// At this point, we can query the Dynatrace API to verify whether our tokens are correct. To avoid excessive requests,
		// we wait at least 5 mins between proves.
		if *t.Timestamp != nil && now.Time.Before((*t.Timestamp).Add(5*time.Minute)) {
			continue
		}

		nowCopy := now
		*t.Timestamp = &nowCopy
		updateCR = true
		ss, err := dtc.GetTokenScopes(t.Value)

		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusUnauthorized {
			oa.SetFailureCondition(t.Type, dynatracev1alpha1.ReasonTokenUnauthorized, fmt.Sprintf("Token on secret %s unauthorized", secretKey))
			continue
		}

		if err != nil {
			oa.SetFailureCondition(t.Type, dynatracev1alpha1.ReasonTokenError, fmt.Sprintf("error when querying token on secret %s: %v", secretKey, err))
			continue
		}

		if !ss.Contains(t.Scope) {
			oa.SetFailureCondition(t.Type, dynatracev1alpha1.ReasonTokenScopeMissing, fmt.Sprintf("Token on secret %s missing scope %s", secretKey, t.Scope))
			continue
		}

		oa.SetCondition(t.Type, corev1.ConditionTrue, dynatracev1alpha1.ReasonTokenReady, "Ready")
	}

	return dtc, updateCR, nil
}
