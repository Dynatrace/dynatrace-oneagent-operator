package nodes

import (
	"context"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileNodes) determineOneAgentForNode(nodeName string) (*dynatracev1alpha1.OneAgent, error) {
	oneAgentList, err := r.getOneAgentList()
	if err != nil {
		return nil, err
	}

	return r.filterOneAgentFromList(oneAgentList, nodeName), nil
}

func (r *ReconcileNodes) getOneAgentList() (*dynatracev1alpha1.OneAgentList, error) {
	watchNamespace := os.Getenv("POD_NAMESPACE")

	var oneAgentList dynatracev1alpha1.OneAgentList
	err := r.client.List(context.TODO(), &oneAgentList, client.InNamespace(watchNamespace))
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
