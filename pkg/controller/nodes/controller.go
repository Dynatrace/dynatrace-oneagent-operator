package nodes

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	oneagent_utils "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent-utils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Controller handles node changes
type Controller struct {
	kubernetesClient kubernetes.Interface
	config           *rest.Config
	logger           logr.Logger

	// cordonedNodes  map[string]interface{}
	nodesInfoCache map[string]*nodesInfo
}

type nodesInfo struct {
	cordoned     bool
	oneagentName string
	internalIP   string
}

// NewController => returns a new instance of Controller
func NewController(config *rest.Config) *Controller {
	c := &Controller{
		kubernetesClient: kubernetes.NewForConfigOrDie(config),
		config:           config,
		logger:           log.Log.WithName("nodes.controller"),
	}
	nodesInfoCache, err := c.getNodesInfoCache()
	if err != nil {
		c.logger.Error(err, "unable to initialise nodes controller", c)
	}
	c.nodesInfoCache = nodesInfoCache
	return c
}

// ReconcileNodes => checks if node is marked unschedulable or unavailable
// and sends adequate event to dynatrace api
func (c *Controller) ReconcileNodes(nodeName string) error {
	node, err := c.kubernetesClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return c.reconcileCordonedNode(nodeName)
		}
		return err
	}

	if !node.Spec.Unschedulable {
		c.setCordonedStatusForNode(nodeName, false)
		return nil
	}

	if c.getCordonedStatusForNode(nodeName) {
		return nil
	}

	return c.reconcileCordonedNode(nodeName)
}

func (c *Controller) setCordonedStatusForNode(nodeName string, cordoned bool) {
	c.nodesInfoCache[nodeName].cordoned = cordoned
}

func (c *Controller) getCordonedStatusForNode(nodeName string) bool {
	return c.nodesInfoCache[nodeName].cordoned
}

func (c *Controller) reconcileCordonedNode(nodeName string) error {

	nodeInfo, ok := c.nodesInfoCache[nodeName]
	if !ok {
		c.logger.Info("node not found in cache", nodeInfo)
		return nil
	}

	oneAgent, err := c.fetchOneAgent(nodeInfo.oneagentName)
	if err != nil {
		return err
	}

	dtc, err := c.buildDynatraceClient(oneAgent)
	if err != nil {
		return err
	}

	err = c.sendMarkedForTerminationEvent(dtc, nodeInfo.internalIP)
	if err != nil {
		return err
	}

	c.setCordonedStatusForNode(nodeName, true)

	return nil
}

func (c *Controller) getNodesInfoCache() (map[string]*nodesInfo, error) {

	oneAgentList, err := c.fetchOneAgentList()
	if err != nil {
		return nil, err
	}

	nodeList, err := c.kubernetesClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	cache := map[string]*nodesInfo{}
	for _, node := range nodeList.Items {
		cache[node.Name] = &nodesInfo{
			oneagentName: c.determineOneAgent(oneAgentList, node).Name,
			internalIP:   c.getInternalIPForNode(node),
			cordoned:     node.Spec.Unschedulable,
		}
	}
	return cache, nil
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

func (c *Controller) fetchOneAgent(name string) (*dynatracev1alpha1.OneAgent, error) {
	runtimeClient, err := client.New(c.config, client.Options{})
	if err != nil {
		return nil, err
	}

	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, err
	}
	namespacedName := types.NamespacedName{
		Namespace: watchNamespace,
		Name:      name,
	}

	oneagent := &dynatracev1alpha1.OneAgent{}
	err = runtimeClient.Get(context.TODO(), namespacedName, oneagent)

	return oneagent, nil
}

func (c *Controller) fetchOneAgentList() (*dynatracev1alpha1.OneAgentList, error) {
	runtimeClient, err := client.New(c.config, client.Options{})
	if err != nil {
		return nil, err
	}

	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, err
	}

	var oneagentList dynatracev1alpha1.OneAgentList
	err = runtimeClient.List(context.TODO(), &client.ListOptions{Namespace: watchNamespace}, &oneagentList)
	if err != nil {
		return nil, err
	}

	return &oneagentList, nil
}

func (c *Controller) determineOneAgent(oneagentList *dynatracev1alpha1.OneAgentList,
	node corev1.Node) *dynatracev1alpha1.OneAgent {

	nodeLabels := node.Labels
	for _, oneAgent := range oneagentList.Items {
		if c.isSubset(oneAgent.Spec.NodeSelector, nodeLabels) {
			return &oneAgent
		}
	}

	return nil
}

func (c *Controller) isSubset(child, parent map[string]string) bool {
	if len(child) == 0 && len(parent) == 0 {
		return true
	}
	if len(child) == 0 || len(parent) == 0 {
		return false
	}

	for k, v := range child {
		if w, ok := parent[k]; !ok || v != w {
			return false
		}
	}

	return true
}

func (c *Controller) sendMarkedForTerminationEvent(dtc dtclient.Client, nodeIP string) error {
	entityID, err := dtc.GetEntityIDForIP(nodeIP)
	if err != nil {
		return err
	}

	event := &dtclient.EventData{
		EventType:      dtclient.MarkedForTerminationEvent,
		Source:         "OneAgent Operator",
		Description:    "Kubernetes node cordoned. Node might be drained or terminated.",
		TimeoutMinutes: 20,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	}
	c.logger.Info("sending mark for termination event to dynatrace server", "node", nodeIP)

	return dtc.SendEvent(event)
}

func (c *Controller) buildDynatraceClient(instance *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {
	secret, err := c.getSecret(instance.Spec.Tokens, instance.Namespace)
	if err != nil {
		return nil, err
	}

	var certificateValidation = dtclient.SkipCertificateValidation(instance.Spec.SkipCertCheck)
	apiToken, _ := oneagent_utils.ExtractToken(secret, oneagent_utils.DynatraceApiToken)
	paasToken, _ := oneagent_utils.ExtractToken(secret, oneagent_utils.DynatraceApiToken)

	return dtclient.NewClient(instance.Spec.ApiUrl, apiToken, paasToken, certificateValidation)
}

func (c *Controller) getSecret(name string, namespace string) (*corev1.Secret, error) {
	secret, err := c.kubernetesClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil, err
	}

	if err = oneagent_utils.VerifySecret(secret); err != nil {
		return nil, err
	}

	return secret, nil
}
