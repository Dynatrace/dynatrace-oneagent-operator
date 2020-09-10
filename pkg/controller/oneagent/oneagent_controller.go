package oneagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"k8s.io/apimachinery/pkg/api/resource"
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
const annotationTemplateHash = "internal.oneagent.dynatrace.com/template-hash"

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
		logger:    logger,
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
	logger.Info("Reconciling OneAgent")

	instance := r.instance.DeepCopyObject().(dynatracev1alpha1.BaseOneAgentDaemonSet)

	// Using the apiReader, which does not use caching to prevent a possible race condition where an old version of
	// the OneAgent object is returned from the cache, but it has already been modified on the cluster side
	if err := r.apiReader.Get(context.Background(), request.NamespacedName, instance); k8serrors.IsNotFound(err) {
		// Request object not dsActual, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	rec := reconciliation{log: logger, instance: instance, requeueAfter: 30 * time.Minute}
	r.reconcileImpl(&rec)

	if rec.err != nil {
		if rec.update || instance.GetOneAgentStatus().SetPhaseOnError(rec.err) {
			if errClient := r.updateCR(instance); errClient != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", rec.err, errClient)
			}
		}

		var serr dtclient.ServerError
		if ok := errors.As(rec.err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			logger.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}

		return reconcile.Result{}, rec.err
	}

	if rec.update {
		if err := r.updateCR(instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: rec.requeueAfter}, nil
}

type reconciliation struct {
	log      logr.Logger
	instance dynatracev1alpha1.BaseOneAgentDaemonSet

	// If update is true, then changes on instance will be sent to the Kubernetes API.
	//
	// Additionally, if err is not nil, then the Reconciliation will fail with its value. Unless it's a Too Many
	// Requests HTTP error from the Dynatrace API, on which case, a reconciliation is requeued after one minute delay.
	//
	// If err is nil, then a reconciliation is requeued after requeueAfter.
	err          error
	update       bool
	requeueAfter time.Duration
}

func (rec *reconciliation) Error(err error) bool {
	if err == nil {
		return false
	}
	rec.err = err
	return true
}

func (rec *reconciliation) Update(upd bool, d time.Duration, cause string) bool {
	if !upd {
		return false
	}
	rec.log.Info("Updating OneAgent CR", "cause", cause)
	rec.update = true
	rec.requeueAfter = d
	return true
}

func (r *ReconcileOneAgent) reconcileImpl(rec *reconciliation) {
	if err := validate(rec.instance); rec.Error(err) {
		return
	}

	dtc, upd, err := r.dtcReconciler.Reconcile(context.Background(), rec.instance)
	rec.Update(upd, 5*time.Minute, "Token conditions updated")
	if rec.Error(err) {
		return
	}

	if rec.instance.GetOneAgentSpec().EnableIstio {
		if upd, err := r.istioController.ReconcileIstio(rec.instance, dtc); err != nil {
			// If there are errors log them, but move on.
			rec.log.Info("Istio: failed to reconcile objects", "error", err)
		} else if upd {
			rec.log.Info("Istio: objects updated")
			rec.requeueAfter = 30 * time.Second
			return
		}
	}

	upd = utils.SetUseImmutableImageStatus(rec.instance, r.logger, dtc)
	if rec.Update(upd, 5*time.Second, "checked cluster version") {
		return
	}

	if rec.instance.GetOneAgentStatus().UseImmutableImage && rec.instance.GetOneAgentSpec().CustomPullSecret == "" {
		err = r.ReconcilePullSecret(rec.instance, rec.log)
		if rec.Error(err) {
			return
		}
	}

	upd, err = r.reconcileRollout(rec.log, rec.instance, dtc)
	if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Rollout reconciled") {
		return
	}

	upd, err = r.reconcileInstanceStatuses(rec.log, rec.instance, dtc)
	if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Instance statuses reconciled") {
		return
	}

	if rec.instance.GetOneAgentSpec().DisableAgentUpdate {
		rec.log.Info("Automatic oneagent update is disabled")
		return
	}

	upd, err = r.reconcileVersion(rec.log, rec.instance, dtc)
	if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Versions reconciled") {
		return
	}

	// Finally we have to determine the correct non error phase
	if upd, err = r.determineOneAgentPhase(rec.instance); !rec.Error(err) {
		rec.Update(upd, 5*time.Minute, "Phase change")
	}
}

func (r *ReconcileOneAgent) reconcileRollout(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired, err := newDaemonSetForCR(logger, instance)
	if err != nil {
		return false, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	dsActual := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dsDesired.Name, Namespace: dsDesired.Namespace}, dsActual)
	if err != nil && k8serrors.IsNotFound(err) {
		logger.Info("Creating new daemonset")
		if err = r.client.Create(context.TODO(), dsDesired); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else if hasDaemonSetChanged(dsDesired, dsActual) {
		logger.Info("Updating existing daemonset")
		if err = r.client.Update(context.TODO(), dsDesired); err != nil {
			return false, err
		}
	}

	if instance.GetOneAgentStatus().Version == "" {
		if instance.GetOneAgentStatus().UseImmutableImage && instance.GetOneAgentSpec().Image == "" {
			if instance.GetOneAgentSpec().AgentVersion == "" {
				latest, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
				if err != nil {
					return false, fmt.Errorf("failed to get desired version: %w", err)
				}
				instance.GetOneAgentStatus().Version = latest
			} else {
				instance.GetOneAgentStatus().Version = instance.GetOneAgentSpec().AgentVersion
			}
		} else {
			desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
			if err != nil {
				return false, fmt.Errorf("failed to get desired version: %w", err)
			}

			logger.Info("Updating version on OneAgent instance")
			instance.GetOneAgentStatus().Version = desired
		}

		instance.GetOneAgentStatus().SetPhase(dynatracev1alpha1.Deploying)
		updateCR = true
	}

	if instance.GetOneAgentStatus().Tokens != utils.GetTokensName(instance) {
		instance.GetOneAgentStatus().Tokens = utils.GetTokensName(instance)
		updateCR = true
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) ReconcilePullSecret(instance dynatracev1alpha1.BaseOneAgent, log logr.Logger) error {
	var tkns corev1.Secret
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: utils.GetTokensName(instance), Namespace: instance.GetNamespace()}, &tkns); err != nil {
		return fmt.Errorf("failed to query tokens: %w", err)
	}
	pullSecretData, err := utils.GeneratePullSecretData(r.client, instance, &tkns)
	if err != nil {
		return fmt.Errorf("failed to generate pull secret data: %w", err)
	}
	err = utils.CreateOrUpdateSecretIfNotExists(r.client, r.client, instance.GetName()+"-pull-secret", instance.GetNamespace(), pullSecretData, corev1.SecretTypeDockerConfigJson, log)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return nil
}

func (r *ReconcileOneAgent) getPods(instance *dynatracev1alpha1.BaseOneAgentDaemonSet) ([]corev1.Pod, []client.ListOption, error) {
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*instance).GetNamespace()),
		client.MatchingLabels(buildLabels((*instance).GetName())),
	}
	err := r.client.List(context.TODO(), podList, listOps...)
	return podList.Items, listOps, err
}

func (r *ReconcileOneAgent) updateCR(instance dynatracev1alpha1.BaseOneAgentDaemonSet) error {
	instance.GetOneAgentStatus().UpdatedTimestamp = metav1.Now()
	return r.client.Status().Update(context.TODO(), instance)
}

func newDaemonSetForCR(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet) (*appsv1.DaemonSet, error) {
	podSpec := newPodSpecForCR(instance, logger)
	selectorLabels := buildLabels(instance.GetName())
	mergedLabels := mergeLabels(instance.GetOneAgentSpec().Labels, selectorLabels)

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Labels:      mergedLabels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: mergedLabels},
				Spec:       podSpec,
			},
		},
	}

	dsHash, err := generateDaemonSetHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[annotationTemplateHash] = dsHash

	return ds, nil
}

func newPodSpecForCR(instance dynatracev1alpha1.BaseOneAgentDaemonSet, logger logr.Logger) corev1.PodSpec {
	p := corev1.PodSpec{}
	trueVar := true

	sa := "dynatrace-oneagent"
	if instance.GetOneAgentSpec().ServiceAccountName != "" {
		sa = instance.GetOneAgentSpec().ServiceAccountName
	}

	resources := instance.GetOneAgentSpec().Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	args := instance.GetOneAgentSpec().Args
	if instance.GetOneAgentSpec().Proxy != nil && (instance.GetOneAgentSpec().Proxy.ValueFrom != "" || instance.GetOneAgentSpec().Proxy.Value != "") {
		args = append(args, "--set-proxy=$(https_proxy)")
	}

	if instance.GetOneAgentSpec().NetworkZone != "" {
		args = append(args, fmt.Sprintf("--set-network-zone=%s", instance.GetOneAgentSpec().NetworkZone))
	}

	if _, ok := instance.(*dynatracev1alpha1.OneAgentIM); ok {
		args = append(args, "--set-infra-only=true")
	}

	args = append(args, "--set-host-property=OperatorVersion="+version.Version)

	// K8s 1.18+ is expected to drop the "beta.kubernetes.io" labels in favor of "kubernetes.io" which was added on K8s 1.14.
	// To support both older and newer K8s versions we use node affinity.

	p = corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            args,
			Env:             nil,
			Image:           "",
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
			Resources: resources,
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

	if instance.GetOneAgentStatus().UseImmutableImage {
		err := preparePodSpecImmutableImage(&p, instance)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v2")
		}
	} else {
		err := preparePodSpecInstaller(&p, instance)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v1")
		}
	}

	return p
}

func preparePodSpecInstaller(p *corev1.PodSpec, instance dynatracev1alpha1.BaseOneAgentDaemonSet) error {
	img := "docker.io/dynatrace/oneagent:latest"
	envVarImg := os.Getenv("RELATED_IMAGE_DYNATRACE_ONEAGENT")

	if instance.GetOneAgentSpec().Image != "" {
		img = instance.GetOneAgentSpec().Image
	} else if envVarImg != "" {
		img = envVarImg
	}

	p.Containers[0].Image = img
	p.Containers[0].Env = prepareEnvVars(instance)
	return nil
}

func preparePodSpecImmutableImage(p *corev1.PodSpec, instance dynatracev1alpha1.BaseOneAgentDaemonSet) error {
	pullSecretName := instance.GetName() + "-pull-secret"
	if instance.GetOneAgentSpec().CustomPullSecret != "" {
		pullSecretName = instance.GetOneAgentSpec().CustomPullSecret
	}

	p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})

	i, err := utils.BuildOneAgentImage(instance.GetSpec().APIURL, instance.GetOneAgentSpec().AgentVersion)
	if err != nil {
		return err
	}
	p.Containers[0].Image = i

	return nil
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

func hasDaemonSetChanged(a, b *appsv1.DaemonSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func generateDaemonSetHash(ds *appsv1.DaemonSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[annotationTemplateHash]
	}
	return ""
}

func (r *ReconcileOneAgent) reconcileInstanceStatuses(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	pods, listOpts, err := r.getPods(&instance)
	if err != nil {
		handlePodListError(logger, err, listOpts)
	}

	instanceStatuses, err := getInstanceStatuses(pods, dtc, instance)
	if err != nil {
		if instanceStatuses == nil || len(instanceStatuses) <= 0 {
			return false, err
		}
	}

	if instance.GetOneAgentStatus().Instances == nil || !reflect.DeepEqual(instance.GetOneAgentStatus().Instances, instanceStatuses) {
		instance.GetOneAgentStatus().Instances = instanceStatuses
		return true, err
	}

	return false, err
}

func getInstanceStatuses(pods []corev1.Pod, dtc dtclient.Client, instance dynatracev1alpha1.BaseOneAgentDaemonSet) (map[string]dynatracev1alpha1.OneAgentInstance, error) {
	instanceStatuses := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		instanceStatus := dynatracev1alpha1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			if err = handleAgentVersionForIPError(err, ver, instance, pod, &instanceStatus); err != nil {
				return instanceStatuses, err
			}
		} else {
			instanceStatus.Version = ver
		}
		instanceStatuses[pod.Spec.NodeName] = instanceStatus
	}
	return instanceStatuses, nil
}
