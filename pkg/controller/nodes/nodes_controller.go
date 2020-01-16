package nodes

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Add creates a new Nodes Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return NewController(mgr.GetClient(), utils.BuildDynatraceClient, log.Log.WithName("nodes.controller"))
}

// add adds a new NodesController to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodes-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Nodes
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// NewController => returns a new instance of Controller
func NewController(client client.Client, dtcFunc utils.DynatraceClientFunc, logger logr.Logger) *ReconcileNodes {
	return &ReconcileNodes{
		client:              client,
		logger:              logger,
		nodeCordonedStatus:  make(map[string]bool),
		dynatraceClientFunc: dtcFunc,
	}
}

// ReconcileNodes reconciles a Nodes object
type ReconcileNodes struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client              client.Client
	logger              logr.Logger
	nodeCordonedStatus  map[string]bool
	dynatraceClientFunc utils.DynatraceClientFunc
}

// Reconcile reads that state of the cluster for a Nodes object and makes changes based on the state read
// and what is in the Nodes.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodes) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, r.ReconcileNodes(request.Name)
}

// ReconcileNodes => checks if node is marked unschedulable or unavailable
// and sends adequate event to dynatrace api
func (r *ReconcileNodes) ReconcileNodes(nodeName string) error {
	var node corev1.Node
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: nodeName}, &node)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.reconcileCordonedNode(nodeName)
		}
		return err
	}

	if !node.Spec.Unschedulable {
		r.nodeCordonedStatus[nodeName] = false
		return nil
	}

	return r.reconcileCordonedNode(nodeName)
}

func (r *ReconcileNodes) reconcileCordonedNode(nodeName string) error {
	if isCordoned, ok := r.nodeCordonedStatus[nodeName]; ok && isCordoned {
		return nil
	}

	oneAgent, err := r.determineOneAgentForNode(nodeName)
	if err != nil {
		return err
	}

	if oneAgent == nil { // If no OneAgent object has been found for node, do nothing.
		return nil
	}

	dtc, err := r.dynatraceClientFunc(r.client, oneAgent)
	if err != nil {
		return err
	}

	err = r.sendMarkedForTerminationEvent(dtc, oneAgent.Status.Instances[nodeName].IPAddress)
	if err != nil {
		return err
	}

	r.nodeCordonedStatus[nodeName] = true

	return nil
}

func (r *ReconcileNodes) determineOneAgentForNode(nodeName string) (*dynatracev1alpha1.OneAgent, error) {
	oneAgentList, err := r.getOneAgentList()
	if err != nil {
		return nil, err
	}

	return r.filterOneAgentFromList(oneAgentList, nodeName), nil
}

func (r *ReconcileNodes) getOneAgentList() (*dynatracev1alpha1.OneAgentList, error) {
	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, err
	}

	var oneAgentList dynatracev1alpha1.OneAgentList
	err = r.client.List(context.TODO(), &oneAgentList, client.InNamespace(watchNamespace))
	if err != nil {
		return nil, err
	}

	return &oneAgentList, nil
}

func (r *ReconcileNodes) filterOneAgentFromList(oneAgentList *dynatracev1alpha1.OneAgentList,
	nodeName string) *dynatracev1alpha1.OneAgent {

	for _, oneAgent := range oneAgentList.Items {
		items := oneAgent.Status.Instances
		if _, ok := items[nodeName]; ok {
			return &oneAgent
		}
	}

	return nil
}

func (r *ReconcileNodes) sendMarkedForTerminationEvent(dtc dtclient.Client, nodeIP string) error {
	entityID, err := dtc.GetEntityIDForIP(nodeIP)
	if err != nil {
		return err
	}

	tenMinutesAgoInMillis := r.makeEventStartTimestamp(time.Now())
	event := &dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "OneAgent Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: tenMinutesAgoInMillis,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	}
	r.logger.Info("sending mark for termination event to dynatrace server", "node", nodeIP)

	return dtc.SendEvent(event)
}

func (r *ReconcileNodes) makeEventStartTimestamp(start time.Time) uint64 {
	backTime := time.Minute * time.Duration(-10)
	tenMinutesAgo := start.Add(backTime).UnixNano()

	return uint64(tenMinutesAgo) / uint64(time.Millisecond)
}
