package nodes

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	oneagent_utils "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent-utils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Controller handles node changes
type Controller struct {
	kubernetesClient   kubernetes.Interface
	config             *rest.Config
	logger             logr.Logger
	nodeCordonedStatus map[string]bool
}

// NewController => returns a new instance of Controller
func NewController(config *rest.Config) *Controller {
	c := &Controller{
		kubernetesClient:   kubernetes.NewForConfigOrDie(config),
		config:             config,
		logger:             log.Log.WithName("nodes.controller"),
		nodeCordonedStatus: make(map[string]bool),
	}

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

	dtc, err := c.buildDynatraceClient(oneAgent)
	if err != nil {
		return err
	}

	err = c.sendMarkedForTerminationEvent(dtc, oneAgent.Status.Items[nodeName].IPAddress)
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
	runtimeClient, err := client.New(c.config, client.Options{})
	if err != nil {
		return nil, err
	}

	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return nil, err
	}

	var oneAgentList dynatracev1alpha1.OneAgentList
	err = runtimeClient.List(context.TODO(), &client.ListOptions{Namespace: watchNamespace}, &oneAgentList)
	if err != nil {
		return nil, err
	}

	return &oneAgentList, nil
}

func (c *Controller) filterOneAgentFromList(oneAgentList *dynatracev1alpha1.OneAgentList,
	nodeName string) *dynatracev1alpha1.OneAgent {

	for _, oneAgent := range oneAgentList.Items {
		items := oneAgent.Status.Items
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

	tenMinutesAgoInMillis := uint64(time.Now().Add(-10*time.Minute).UnixNano() / 1_000_000)
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
