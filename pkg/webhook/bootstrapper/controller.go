package bootstrapper

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/webhook"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/go-logr/logr"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

const webhookName = "dynatrace-oneagent-webhook"
const certsDir = "/mnt/webhook-certs"

// AddToManager creates a new OneAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddToManager(mgr manager.Manager) error {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	return add(mgr, &ReconcileWebhook{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		namespace: ns,
		logger:    log.Log.WithName("webhook.controller"),
	})
}

// add adds a new OneAgentController to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileWebhook) error {
	// Create a new controller
	c, err := controller.New("webhook-bootstrapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ch := make(chan event.GenericEvent, 10)

	if err = c.Watch(&source.Channel{Source: ch}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Create an artificial request
	ch <- event.GenericEvent{
		Meta: &metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: r.namespace,
		},
	}

	return nil
}

// ReconcileWebhook reconciles the webhook
type ReconcileWebhook struct {
	client    client.Client
	scheme    *runtime.Scheme
	logger    logr.Logger
	namespace string
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWebhook) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling webhook", "namespace", request.Namespace, "name", request.Name)

	ctx := context.TODO()

	if err := r.reconcileService(ctx, r.logger); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile service: %w", err)
	}

	rootCerts, err := r.reconcileCerts(ctx, r.logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile certificates: %w", err)
	}

	if err := r.reconcileWebhookConfig(ctx, r.logger, rootCerts); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile webhook configuration: %w", err)
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ReconcileWebhook) reconcileService(ctx context.Context, log logr.Logger) error {
	log.Info("Reconciling Service...")

	expected := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"dynatrace.com/operator":                    "oneagent",
				"internal.oneagent.dynatrace.com/component": "webhook",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"internal.oneagent.dynatrace.com/component": "webhook",
				"internal.oneagent.dynatrace.com/app":       "webhook",
			},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       443,
				TargetPort: intstr.FromString("server-port"),
			}},
		},
	}

	var svc corev1.Service

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName, Namespace: r.namespace}, &svc)
	if k8serrors.IsNotFound(err) {
		log.Info("Service doesn't exist, creating...")
		if err = r.client.Create(ctx, &expected); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileWebhook) reconcileCerts(ctx context.Context, log logr.Logger) ([]byte, error) {
	log.Info("Reconciling certificates...")

	if data, err := ioutil.ReadFile(certsDir + "/tls.crt"); err == nil {
		return data, nil
	}

	var cs certs

	domain := fmt.Sprintf("%s.%s.svc", webhookName, r.namespace)

	log.Info("Generating root certificates...")

	if err := cs.generateRootCerts(certsDir, domain); err != nil {
		return nil, err
	}

	log.Info("Generating server certificates...")

	if err := cs.generateServerCerts(certsDir, domain); err != nil {
		return nil, err
	}

	return cs.rootPublicCertPEM, nil
}

func (r *ReconcileWebhook) reconcileWebhookConfig(ctx context.Context, log logr.Logger, rootCerts []byte) error {
	log.Info("Reconciling MutatingWebhookConfiguration...")

	scope := admissionregistrationv1beta1.NamespacedScope
	path := "/inject"
	expected := admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"dynatrace.com/operator":                    "oneagent",
				"internal.oneagent.dynatrace.com/component": "webhook",
			},
		},
		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{{
			Name:                    "webhook.oneagent.dynatrace.com",
			AdmissionReviewVersions: []string{"v1beta1"},
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
					Scope:       &scope,
				},
			}},
			NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      webhook.LabelInstance,
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Name:      webhookName,
					Namespace: r.namespace,
					Path:      &path,
				},
				CABundle: rootCerts,
			},
		}},
	}

	var cfg admissionregistrationv1beta1.MutatingWebhookConfiguration

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("MutatingWebhookConfiguration doesn't exist, creating...")
		if err = r.client.Create(ctx, &expected); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	if reflect.DeepEqual(&expected.Webhooks, &cfg.Webhooks) {
		return nil
	}

	log.Info("MutatingWebhookConfiguration is outdated, updating...")
	cfg.Webhooks = expected.Webhooks
	return r.client.Update(ctx, &cfg)
}
