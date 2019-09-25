package nodes

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var cordonedNodes = make(map[string]bool)

// Controller => controller instance for nodes
type Controller struct {
	logger           logr.Logger
	restConfig       *rest.Config
	dynatraceClient  dtclient.Client
	kubernetesClient kubernetes.Interface
}

func NewController(config *rest.Config) *Controller {
	c := &Controller{
		restConfig: config,
		logger:     log.Log.WithName("nodes.controller"),
	}
	c.kubernetesClient = kubernetes.NewForConfigOrDie(c.restConfig)

	return c
}

func (c *Controller) ReconcileNodes(nodeName string) {

	_, _ = c.determineCustomResource(nodeName)

}

func (c *Controller) determineCustomResource(nodeName string) (*dynatracev1alpha1.OneAgent, error) {

	runtimeClient, err := runtimeclient.New(c.restConfig, runtimeclient.Options{})
	if err != nil {
		return nil, err
	}

	var oneagentList *dynatracev1alpha1.OneAgentList
	err = runtimeClient.List(context.TODO(), nil, oneagentList)
	if err != nil {
		return nil, err
	}

	node, err := c.kubernetesClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	nodeLabels := node.Labels

	for _, oneagent := range oneagentList.Items {
		if c.isSubset(oneagent.Labels, nodeLabels) {
			return &oneagent, nil
		}
	}

	return nil, err
}

func (c *Controller) isSubset(child, parent map[string]string) bool {
	for k, v := range child {
		if w, ok := parent[k]; !ok || v != w {
			return false
		}
	}

	return true
}

func intialiseDtClient(instance *dynatracev1alpha1.OneAgent) {}
func reconcile()                                             {}

func (c *Controller) reconcileCordonedNode(instance *dynatracev1alpha1.OneAgent) error {

	listops := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(instance.Spec.NodeSelector).String(),
	}
	nodes, err := c.kubernetesClient.CoreV1().Nodes().Get("name", metav1.GetOptions{})
	if err != nil {
		c.logger.Info("failed to list nodes", "with options", listops)
		return err
	}
	print(nodes)
	// for _, node := range nodes.Items {
	// 	cordoned := node.Spec.Unschedulable
	// 	nodeInternalIP := c.getInternalIPForNode(node)
	// 	reported, ok := cordonedNodes[nodeInternalIP]

	// 	if !cordoned {
	// 		delete(cordonedNodes, nodeInternalIP)
	// 	} else if !reported || !ok {
	// 		err := c.notifyDynatraceAboutMarkForTerminationEvent(nodeInternalIP)
	// 		if err != nil {
	// 			c.logger.Info("failed to send mark for termination notification to dynatrace", "error", err)
	// 			cordonedNodes[nodeInternalIP] = false
	// 		} else {
	// 			cordonedNodes[nodeInternalIP] = true
	// 		}
	// 	}
	// }

	return nil
}

func (c *Controller) getInternalIPForNode(node corev1.Node) string {

	addresses := node.Status.Addresses
	if len(addresses) == 0 {
		return ""
	}
	for _, addr := range addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func (c *Controller) notifyDynatraceAboutMarkForTerminationEvent(nodeIP string) error {
	entityID, err := c.dynatraceClient.GetEntityIDForIP(nodeIP)
	if err != nil {
		return err
	}

	event := &dtclient.EventData{
		EventType:             dtclient.MarkForTerminationEvent,
		Source:                "Dynatrace OneAgent Operator",
		AnnotationDescription: "Kubernetes node marked unschedulable. Node is likely being drained.",
		TimeoutMinutes:        20,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	}

	return c.dynatraceClient.SendEvent(event)
}
