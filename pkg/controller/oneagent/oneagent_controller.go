package oneagent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/version"

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
	return add(mgr, NewOneAgentReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetConfig(),
		log.Log.WithName("oneagent.controller"),
		utils.BuildDynatraceClient,
		&dynatracev1alpha1.OneAgent{}))
}

// NewOneAgentReconciler initialises a new ReconcileOneAgent instance
func NewOneAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, logger logr.Logger,
	dtcFunc utils.DynatraceClientFunc, instance dynatracev1alpha1.BaseOneAgentDaemonSet) *ReconcileOneAgent {
	return &ReconcileOneAgent{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		config:    config,
		logger:    log.Log.WithName("oneagent.controller"),
		dtcReconciler: &utils.DynatraceClientReconciler{
			DynatraceClientFunc: dtcFunc,
			Client:              client,
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
		istioController: istio.NewController(config, scheme),
		instance:        instance,
	}
}

// add adds a new OneAgentController to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileOneAgent) error {
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

	dtcReconciler   *utils.DynatraceClientReconciler
	istioController *istio.Controller
	instance        dynatracev1alpha1.BaseOneAgentDaemonSet
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", request.Namespace, "name", request.Name)
	logger.Info("reconciling oneagent")

	instance := r.instance.DeepCopyObject().(dynatracev1alpha1.BaseOneAgentDaemonSet)
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
		updateCR := instance.GetOneAgentStatus().SetPhaseOnError(err)
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
	dtc, updateCR, err := r.dtcReconciler.Reconcile(context.TODO(), instance)
	if instance.GetOneAgentStatus().SetPhaseOnError(err) || updateCR {
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

	if instance.GetOneAgentSpec().EnableIstio {
		if upd, err := r.istioController.ReconcileIstio(instance, dtc); err != nil {
			// If there are errors log them, but move on.
			logger.Info("istio: failed to reconcile objects", "error", err)
		} else if upd {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	updateCR, err = r.reconcileRollout(logger, instance, dtc)
	if instance.GetOneAgentStatus().SetPhaseOnError(err) || updateCR {
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
		return reconcile.Result{}, err
	}

	if instance.GetOneAgentSpec().DisableAgentUpdate {
		logger.Info("automatic oneagent update is disabled")
		return reconcile.Result{}, nil
	}

	updateCR, err = r.reconcileVersion(logger, instance, dtc)
	if instance.GetOneAgentStatus().SetPhaseOnError(err) || updateCR {
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

func (r *ReconcileOneAgent) reconcileRollout(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
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

	if instance.GetOneAgentStatus().Version == "" {
		desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			return updateCR, fmt.Errorf("failed to get desired version: %w", err)
		}

		instance.GetOneAgentStatus().Version = desired
		instance.GetOneAgentStatus().SetPhase(dynatracev1alpha1.Deploying)
		updateCR = true
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) determineOneAgentPhase(instance dynatracev1alpha1.BaseOneAgentDaemonSet) (bool, error) {
	var phaseChanged bool
	dsActual := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.GetName(), Namespace: instance.GetNamespace()}, dsActual)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		phaseChanged = instance.GetOneAgentStatus().Phase != dynatracev1alpha1.Error
		instance.GetOneAgentStatus().Phase = dynatracev1alpha1.Error
		return phaseChanged, err
	}

	if dsActual.Status.NumberReady == dsActual.Status.CurrentNumberScheduled {
		phaseChanged = instance.GetOneAgentStatus().Phase != dynatracev1alpha1.Running
		instance.GetOneAgentStatus().Phase = dynatracev1alpha1.Running
	} else {
		phaseChanged = instance.GetOneAgentStatus().Phase != dynatracev1alpha1.Deploying
		instance.GetOneAgentStatus().Phase = dynatracev1alpha1.Deploying
	}

	return phaseChanged, nil
}

func (r *ReconcileOneAgent) reconcileVersion(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	updateCR := false

	// get desired version
	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return false, fmt.Errorf("failed to get desired version: %w", err)
	} else if desired != "" && instance.GetOneAgentStatus().Version != desired {
		logger.Info("new version available", "actual", instance.GetOneAgentStatus().Version, "desired", desired)
		instance.GetOneAgentStatus().Version = desired
		updateCR = true
	}

	// query oneagent pods
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(buildLabels(instance.GetName())),
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
	if (len(instances) > 0 || len(instance.GetOneAgentStatus().Instances) > 0) && !reflect.DeepEqual(instances, instance.GetOneAgentStatus().Instances) {
		logger.Info("oneagent pod instances changed", "status", instance.GetOneAgentStatus())
		updateCR = true
		instance.GetOneAgentStatus().Instances = instances
	}

	var waitSecs uint16 = 300
	if instance.GetOneAgentSpec().WaitReadySeconds != nil {
		waitSecs = *instance.GetOneAgentSpec().WaitReadySeconds
	}

	if len(podsToDelete) > 0 {
		if instance.GetOneAgentStatus().SetPhase(dynatracev1alpha1.Deploying) {
			err := r.updateCR(instance)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to set phase to %s", dynatracev1alpha1.Deploying))
			}
		}
	}

	// restart daemonset
	err = r.deletePods(logger, podsToDelete, buildLabels(instance.GetName()), waitSecs)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) updateCR(instance dynatracev1alpha1.BaseOneAgentDaemonSet) error {
	instance.GetOneAgentStatus().UpdatedTimestamp = metav1.Now()
	return r.client.Status().Update(context.TODO(), instance)
}

func newDaemonSetForCR(instance dynatracev1alpha1.BaseOneAgentDaemonSet) *appsv1.DaemonSet {
	podSpec := newPodSpecForCR(instance)
	selectorLabels := buildLabels(instance.GetName())
	mergedLabels := mergeLabels(instance.GetOneAgentSpec().Labels, selectorLabels)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetName(),
			Namespace: instance.GetNamespace(),
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

func newPodSpecForCR(instance dynatracev1alpha1.BaseOneAgentDaemonSet) corev1.PodSpec {
	trueVar := true

	envVarImg := os.Getenv("RELATED_IMAGE_DYNATRACE_ONEAGENT")
	img := "docker.io/dynatrace/oneagent:latest"
	if instance.GetOneAgentSpec().Image != "" {
		img = instance.GetOneAgentSpec().Image
	} else if envVarImg != "" {
		img = envVarImg
	}

	sa := "dynatrace-oneagent"
	if instance.GetOneAgentSpec().ServiceAccountName != "" {
		sa = instance.GetOneAgentSpec().ServiceAccountName
	}

	args := instance.GetOneAgentSpec().Args
	if instance.GetOneAgentSpec().Proxy != nil && (instance.GetOneAgentSpec().Proxy.ValueFrom != "" || instance.GetOneAgentSpec().Proxy.Value != "") {
		args = append(instance.GetOneAgentSpec().Args, "--set-proxy=$(https_proxy)")
	}

	if _, ok := instance.(*dynatracev1alpha1.OneAgentIM); ok {
		args = append(args, "--set-infra-only=true")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

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
			Resources: instance.GetOneAgentSpec().Resources,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &trueVar,
			},
			VolumeMounts: prepareVolumeMounts(instance),
		}},
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            true,
		NodeSelector:       instance.GetOneAgentSpec().NodeSelector,
		PriorityClassName:  instance.GetOneAgentSpec().PriorityClassName,
		ServiceAccountName: sa,
		Tolerations:        instance.GetOneAgentSpec().Tolerations,
		DNSPolicy:          instance.GetOneAgentSpec().DNSPolicy,
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "beta.kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
									Key:      "beta.kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
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

func prepareVolumes(instance dynatracev1alpha1.BaseOneAgentDaemonSet) []corev1.Volume {
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

	if instance.GetOneAgentSpec().TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "certs",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.GetOneAgentSpec().TrustedCAs,
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

func prepareVolumeMounts(instance dynatracev1alpha1.BaseOneAgentDaemonSet) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "host-root",
			MountPath: "/mnt/root",
		},
	}

	if instance.GetOneAgentSpec().TrustedCAs != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "certs",
			MountPath: "/mnt/dynatrace/certs",
		})
	}

	return volumeMounts
}

func prepareEnvVars(instance dynatracev1alpha1.BaseOneAgentDaemonSet) []corev1.EnvVar {
	var token, installerURL, skipCert, proxy *corev1.EnvVar

	reserved := map[string]**corev1.EnvVar{
		"ONEAGENT_INSTALLER_TOKEN":           &token,
		"ONEAGENT_INSTALLER_SCRIPT_URL":      &installerURL,
		"ONEAGENT_INSTALLER_SKIP_CERT_CHECK": &skipCert,
		"https_proxy":                        &proxy,
	}

	var envVars []corev1.EnvVar

	for i := range instance.GetOneAgentSpec().Env {
		if p := reserved[instance.GetOneAgentSpec().Env[i].Name]; p != nil {
			*p = &instance.GetOneAgentSpec().Env[i]
			continue
		}
		envVars = append(envVars, instance.GetOneAgentSpec().Env[i])
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
			Value: fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=$(ONEAGENT_INSTALLER_TOKEN)&arch=x86&flavor=default", instance.GetOneAgentSpec().APIURL),
		}
	}

	if skipCert == nil {
		skipCert = &corev1.EnvVar{
			Name:  "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
			Value: strconv.FormatBool(instance.GetOneAgentSpec().SkipCertCheck),
		}
	}

	env := []corev1.EnvVar{*token, *installerURL, *skipCert}

	if proxy == nil {
		if instance.GetOneAgentSpec().Proxy != nil {
			if instance.GetOneAgentSpec().Proxy.ValueFrom != "" {
				env = append(env, corev1.EnvVar{
					Name: "https_proxy",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: instance.GetOneAgentSpec().Proxy.ValueFrom},
							Key:                  "proxy",
						},
					},
				})
			} else if instance.GetOneAgentSpec().Proxy.Value != "" {
				env = append(env, corev1.EnvVar{
					Name:  "https_proxy",
					Value: instance.GetOneAgentSpec().Proxy.Value,
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
