package nodes

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	oneagentutils "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagentutils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatraceclient"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Controller handles node changes
type Controller struct {
	client              client.Client
	logger              logr.Logger
	nodeCordonedStatus  map[string]bool
	dynatraceClientFunc oneagentutils.DynatraceClientFunc
}

// NewController => returns a new instance of Controller
func NewController(client client.Client, dtcFunc oneagentutils.DynatraceClientFunc) *Controller {
	return &Controller{
		client:              client,
		logger:              log.Log.WithName("nodes.controller"),
		nodeCordonedStatus:  make(map[string]bool),
		dynatraceClientFunc: dtcFunc,
	}
}

// ReconcileNodes => checks if node is marked unschedulable or unavailable
// and sends adequate event to dynatrace api
func (c *Controller) ReconcileNodes(nodeName string) error {
	var node corev1.Node
	err := c.client.Get(context.TODO(), client.ObjectKey{Name: nodeName}, &node)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.reconcileCordonedNode(nodeName)
		}
		return err
	}

	if !node.Spec.Unschedulable {
		c.nodeCordonedStatus[nodeName] = false
		return nil
	}

	return c.reconcileCordonedNode(nodeName)
}

func (c *Controller) reconcileCordonedNode(nodeName string) error {
	if isCordoned, ok := c.nodeCordonedStatus[nodeName]; ok && isCordoned {
		return nil
	}

	oneAgent, err := c.determineOneAgentForNode(nodeName)
	if err != nil {
		return err
	}

	if oneAgent == nil { // If no OneAgent object has been found for node, do nothing.
		return nil
	}

	dtc, err := c.dynatraceClientFunc(c.client, oneAgent)
	if err != nil {
		return err
	}

	err = c.sendMarkedForTerminationEvent(dtc, oneAgent.Status.Instances[nodeName].IPAddress)
	if err != nil {
		return err
	}

	c.nodeCordonedStatus[nodeName] = true

	return nil
}

func (c *Controller) determineOneAgentForNode(nodeName string) (*dynatracev1alpha1.OneAgent, error) {
	oneAgentList, err := c.getOneAgentList()
	if err != nil {
		return nil, err
	}

	return c.filterOneAgentFromList(oneAgentList, nodeName), nil
}

func (c *Controller) getOneAgentList() (*dynatracev1alpha1.OneAgentList, error) {
	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, err
	}

	var oneAgentList dynatracev1alpha1.OneAgentList
	err = c.client.List(context.TODO(), &oneAgentList, client.InNamespace(watchNamespace))
	if err != nil {
		return nil, err
	}

	return &oneAgentList, nil
}

func (c *Controller) filterOneAgentFromList(oneAgentList *dynatracev1alpha1.OneAgentList,
	nodeName string) *dynatracev1alpha1.OneAgent {

	for _, oneAgent := range oneAgentList.Items {
		items := oneAgent.Status.Instances
		if _, ok := items[nodeName]; ok {
			return &oneAgent
		}
	}

	return nil
}

func (c *Controller) sendMarkedForTerminationEvent(dtc dtclient.Client, nodeIP string) error {
	entityID, err := dtc.GetEntityIDForIP(nodeIP)
	if err != nil {
		return err
	}

	tenMinutesAgoInMillis := c.makeEventStartTimestamp(time.Now())
	event := &dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "OneAgent Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: tenMinutesAgoInMillis,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	}
	c.logger.Info("sending mark for termination event to dynatrace server", "node", nodeIP)

	return dtc.SendEvent(event)
}

func (c *Controller) makeEventStartTimestamp(start time.Time) uint64 {
	backTime := time.Minute * time.Duration(-10)
	tenMinutesAgo := start.Add(backTime).UnixNano()

	return uint64(tenMinutesAgo) / uint64(time.Millisecond)
}
