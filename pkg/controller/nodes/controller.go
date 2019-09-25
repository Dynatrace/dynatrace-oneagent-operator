package nodes

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent-utils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Controller handles node changes
type Controller struct {
	kubernetesClient kubernetes.Interface
	scheme           *runtime.Scheme
	config           *rest.Config
	logger           logr.Logger
}

func NewController(scheme *runtime.Scheme, config *rest.Config) *Controller {
	return &Controller{
		kubernetesClient: kubernetes.NewForConfigOrDie(config),
		scheme:           scheme,
		config:           config,
		logger:           log.Log.WithName("nodes.controller"),
	}
}

func (c *Controller) ReconcileNodes(nodeName string) error {
	node, err := c.kubernetesClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !node.Spec.Unschedulable {
		return nil
	}

	dtc, err := c.buildDynatraceClientForNode(node)
	if err != nil {
		return err
	}

	return c.reconcileCordonedNode(dtc, node)
}

func (c *Controller) reconcileCordonedNode(dtc dtclient.Client, node *corev1.Node) error {
	entityID, err := dtc.GetEntityIDForIP(c.getInternalIPForNode(node))
	if err != nil {
		return err
	}

	event := &dtclient.EventData{
		EventType:             dtclient.MarkForTerminationEvent,
		Source:                "Dynatrace OneAgent Operator",
		AnnotationDescription: "Kubernetes node cordoned. Node might be drained or terminated.",
		TimeoutMinutes:        20,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	}
	c.logger.Info("sending mark for termination event to dynatrace server", "node", node.Name)

	return dtc.SendEvent(event)
}

func (c *Controller) getInternalIPForNode(node *corev1.Node) string {
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

func (c *Controller) buildDynatraceClientForNode(node *corev1.Node) (dtclient.Client, error) {
	oneAgent, err := c.determineCustomResource(node)
	if err != nil {
		return nil, err
	}

	return c.buildDynatraceClient(oneAgent)
}

func (c *Controller) determineCustomResource(node *corev1.Node) (*dynatracev1alpha1.OneAgent, error) {
	runtimeClient, err := client.New(c.config, client.Options{})
	if err != nil {
		return nil, err
	}

	var oneagentList *dynatracev1alpha1.OneAgentList
	err = runtimeClient.List(context.TODO(), nil, oneagentList)
	if err != nil {
		return nil, err
	}

	nodeLabels := node.Labels

	for _, oneAgent := range oneagentList.Items {
		if c.isSubset(oneAgent.Labels, nodeLabels) {
			return &oneAgent, nil
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
