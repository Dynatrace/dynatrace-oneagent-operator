package nodes

import (
	"context"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
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

func (r *ReconcileNodes) findOneAgentByName(oaName string) (*dynatracev1alpha1.OneAgent, error) {
	var oa dynatracev1alpha1.OneAgent
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: oaName, Namespace: r.namespace}, &oa); err != nil {
		return nil, err
	}
	return &oa, nil
}
