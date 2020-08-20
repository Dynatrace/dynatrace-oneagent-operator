package oneagentapm

import (
	"context"
	"errors"
	"fmt"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"

	"github.com/go-logr/logr"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new OneAgentAPM Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	client := mgr.GetClient()
	config := mgr.GetConfig()
	scheme := mgr.GetScheme()

	return add(mgr, &ReconcileOneAgentAPM{
		client:    client,
		apiReader: mgr.GetAPIReader(),
		scheme:    scheme,
		config:    config,
		logger:    log.Log.WithName("oneagentapm.controller"),

		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:          client,
			UpdatePaaSToken: true,
		},
		istioController: istio.NewController(config, scheme),
	})
}

// add adds a new OneAgentAPM Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("oneagentapm-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OneAgentAPM
	return c.Watch(&source.Kind{Type: &dynatracev1alpha1.OneAgentAPM{}}, &handler.EnqueueRequestForObject{})
}

// ReconcileOneAgentAPM reconciles a OneAgentAPM object
type ReconcileOneAgentAPM struct {
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

// Reconcile reads that state of the cluster for a OneAgentAPM object and makes changes based on the state read
// and what is in the OneAgentAPM.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgentAPM) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", request.Namespace, "name", request.Name)
	logger.Info("Reconciling OneAgentAPM")

	instance := &dynatracev1alpha1.OneAgentAPM{}

	// Using the apiReader, which does not use caching to prevent a possible race condition where an old version of
	// the OneAgentAPM object is returned from the cache, but it has already been modified on the cluster side
	if err := r.apiReader.Get(context.TODO(), request.NamespacedName, instance); k8serrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.APIURL == "" {
		return reconcile.Result{}, errors.New(".spec.apiUrl is missing")
	}

	dtc, upd, err := r.dtcReconciler.Reconcile(context.TODO(), instance)

	if upd {
		instance.Status.UpdatedTimestamp = metav1.Now()
		reconcileError := err
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			if reconcileError != nil {
				// If update fails, but previous reconciliation did so too, make sure both errors are logged
				logger.Error(reconcileError, reconcileError.Error())
			}
			return reconcile.Result{}, fmt.Errorf("failed to update OneAgentAPM: %w", err)
		}
	}

	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.EnableIstio {
		if upd, err := r.istioController.ReconcileIstio(instance, dtc); err != nil {
			// If there are errors log them, but move on.
			logger.Info("istio: failed to reconcile objects", "error", err)
		} else if upd {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
}
