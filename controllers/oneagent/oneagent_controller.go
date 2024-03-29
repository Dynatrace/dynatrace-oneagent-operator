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

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/istio"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
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
const annotationImageVersion = "internal.oneagent.dynatrace.com/image-version"
const annotationTemplateHash = "internal.oneagent.dynatrace.com/template-hash"
const defaultUpdateInterval = 15 * time.Minute
const updateEnvVar = "ONEAGENT_OPERATOR_UPDATE_INTERVAL"
const imageProbeInterval = 15 * time.Minute
const oneagentDockerImage = "docker.io/dynatrace/oneagent:latest"
const oneagentRedhatImage = "registry.connect.redhat.com/dynatrace/oneagent:latest"

// Add creates a new OneAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ string) error {
	return add(mgr, NewOneAgentReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetConfig(),
		log.Log.WithName("oneagent.controller"),
		utils.BuildDynatraceClient))
}

// NewOneAgentReconciler initializes a new ReconcileOneAgent instance
func NewOneAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, logger logr.Logger,
	dtcFunc utils.DynatraceClientFunc) *ReconcileOneAgent {
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
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", request.Namespace, "name", request.Name)
	logger.Info("Reconciling OneAgent")

	instance := &dynatracev1alpha1.OneAgent{}

	// Using the apiReader, which does not use caching to prevent a possible race condition where an old version of
	// the OneAgent object is returned from the cache, but it has already been modified on the cluster side
	if err := r.apiReader.Get(ctx, request.NamespacedName, instance); k8serrors.IsNotFound(err) {
		// Request object not dsActual, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	rec := reconciliation{log: logger, instance: instance, requeueAfter: 30 * time.Minute}
	r.reconcileImpl(ctx, &rec)

	if rec.err != nil {
		if rec.update || instance.GetOneAgentStatus().SetPhaseOnError(rec.err) {
			if errClient := r.updateCR(ctx, instance); errClient != nil {
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
		if err := r.updateCR(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: rec.requeueAfter}, nil
}

type reconciliation struct {
	log      logr.Logger
	instance *dynatracev1alpha1.OneAgent

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

func (r *ReconcileOneAgent) reconcileImpl(ctx context.Context, rec *reconciliation) {
	if err := validate(rec.instance); rec.Error(err) {
		return
	}

	dtc, upd, err := r.dtcReconciler.Reconcile(ctx, rec.instance)
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

	rec.Update(utils.SetUseImmutableImageStatus(rec.instance), 5*time.Minute, "UseImmutableImage changed")

	upd, err = r.reconcileImageVersion(ctx, rec.instance, rec.log)
	rec.Update(upd, 5*time.Minute, "ImageVersion updated")
	rec.Error(err)

	if rec.instance.GetOneAgentStatus().UseImmutableImage && rec.instance.GetOneAgentSpec().Image == "" {
		err = r.reconcilePullSecret(ctx, rec.instance, rec.log)
		if rec.Error(err) {
			return
		}
	}

	upd, err = r.reconcileRollout(ctx, rec.log, rec.instance, dtc)
	if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Rollout reconciled") {
		return
	}

	now := metav1.Now()
	updInterval := defaultUpdateInterval
	if val := os.Getenv(updateEnvVar); val != "" {
		x, err := strconv.Atoi(val)
		if err != nil {
			rec.log.Info("Conversion of ONEAGENT_OPERATOR_UPDATE_INTERVAL failed")
		} else {
			updInterval = time.Duration(x) * time.Minute
		}
	}

	if rec.instance.Status.LastUpdateProbeTimestamp == nil || rec.instance.Status.LastUpdateProbeTimestamp.Add(updInterval).Before(now.Time) {
		rec.instance.Status.LastUpdateProbeTimestamp = &now
		rec.Update(true, 5*time.Minute, "updated last update time stamp")

		upd, err = r.reconcileInstanceStatuses(ctx, rec.log, rec.instance, dtc)
		if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Instance statuses reconciled") {
			return
		}

		if rec.instance.GetOneAgentSpec().DisableAgentUpdate {
			rec.log.Info("Automatic oneagent update is disabled")
			return
		}

		upd, err = r.reconcileVersion(ctx, rec.log, rec.instance, dtc)
		if rec.Error(err) || rec.Update(upd, 5*time.Minute, "Versions reconciled") {
			return
		}
	}

	// Finally we have to determine the correct non error phase
	if upd, err = r.determineOneAgentPhase(rec.instance); !rec.Error(err) {
		rec.Update(upd, 5*time.Minute, "Phase change")
	}
}

func (r *ReconcileOneAgent) reconcileRollout(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	updateCR := false

	var kubeSystemNS corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: "kube-system"}, &kubeSystemNS); err != nil {
		return false, fmt.Errorf("failed to query for cluster ID: %w", err)
	}

	// Define a new DaemonSet object
	dsDesired, err := newDaemonSetBuilder(logger, instance, string(kubeSystemNS.UID)).newDaemonSetForCR()
	if err != nil {
		return false, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	dsActual := &appsv1.DaemonSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: dsDesired.Name, Namespace: dsDesired.Namespace}, dsActual)
	if err != nil && k8serrors.IsNotFound(err) {
		logger.Info("Creating new daemonset")
		if err = r.client.Create(ctx, dsDesired); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else if hasDaemonSetChanged(dsDesired, dsActual) {
		logger.Info("Updating existing daemonset")
		if err = r.client.Update(ctx, dsDesired); err != nil {
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

func (r *ReconcileOneAgent) reconcileImageVersion(ctx context.Context, instance *dynatracev1alpha1.OneAgent, log logr.Logger) (bool, error) {
	if !instance.Status.UseImmutableImage || instance.Spec.DisableAgentUpdate {
		return false, nil
	}

	now := metav1.Now()
	if last := instance.Status.LastImageVersionProbeTimestamp; last != nil && last.Add(imageProbeInterval).After(now.Time) {
		return false, nil
	}

	var err error

	image := instance.Spec.Image
	if image == "" {
		if image, err = utils.BuildOneAgentImage(instance.Spec.APIURL, instance.Spec.AgentVersion); err != nil {
			return false, err
		}
	}

	instance.Status.LastImageVersionProbeTimestamp = &now

	psName := instance.Name + "-pull-secret"
	if instance.Spec.CustomPullSecret != "" {
		psName = instance.Spec.CustomPullSecret
	}

	var ps corev1.Secret
	if err = r.client.Get(ctx, client.ObjectKey{Namespace: instance.Namespace, Name: psName}, &ps); err != nil {
		return true, err
	}

	dockerCfg, err := utils.NewDockerConfig(&ps)
	if err != nil {
		return true, err
	}

	ver, err := utils.GetImageVersion(image, dockerCfg)
	if err != nil {
		return true, err
	}

	oldVersion := instance.Status.ImageVersion
	if ver.Version != oldVersion && (oldVersion == "" || isDesiredNewer(oldVersion, ver.Version, log)) {
		log.Info("image update found",
			"oldHash", instance.Status.ImageHash,
			"newHash", ver.Hash,
			"oldVersion", oldVersion,
			"newVersion", ver.Version)

		// Only update hash in case of version changes.
		instance.Status.ImageHash = ver.Hash
		instance.Status.ImageVersion = ver.Version
	}

	return true, nil
}

func (r *ReconcileOneAgent) reconcilePullSecret(ctx context.Context, instance dynatracev1alpha1.BaseOneAgent, log logr.Logger) error {
	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(instance), Namespace: instance.GetNamespace()}, &tkns); err != nil {
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

func (r *ReconcileOneAgent) getPods(ctx context.Context, instance *dynatracev1alpha1.OneAgent) ([]corev1.Pod, []client.ListOption, error) {
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*instance).GetNamespace()),
		client.MatchingLabels(buildLabels((*instance).GetName())),
	}
	err := r.client.List(ctx, podList, listOps...)
	return podList.Items, listOps, err
}

func (r *ReconcileOneAgent) updateCR(ctx context.Context, instance *dynatracev1alpha1.OneAgent) error {
	instance.GetOneAgentStatus().UpdatedTimestamp = metav1.Now()
	return r.client.Status().Update(ctx, instance)
}

func (r *ReconcileOneAgent) reconcileInstanceStatuses(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	pods, listOpts, err := r.getPods(ctx, instance)
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

func getInstanceStatuses(pods []corev1.Pod, dtc dtclient.Client, instance *dynatracev1alpha1.OneAgent) (map[string]dynatracev1alpha1.OneAgentInstance, error) {
	instanceStatuses := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		instanceStatus := dynatracev1alpha1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			if err = handleAgentVersionForIPError(err, instance, pod, &instanceStatus); err != nil {
				return instanceStatuses, err
			}
		} else {
			instanceStatus.Version = ver
		}
		instanceStatuses[pod.Spec.NodeName] = instanceStatus
	}
	return instanceStatuses, nil
}
