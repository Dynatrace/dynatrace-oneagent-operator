package nodes

import (
	"testing"

	apis "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	oneagent_utils "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent-utils"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
}

func TestDetermineCustomResource(t *testing.T) {
	node := corev1.Node{Spec: corev1.NodeSpec{}}
	node.Name = "node_1"
	nodesController := &Controller{}

	{
		oneAgentStatus := dynatracev1alpha1.OneAgentStatus{
			Items: map[string]dynatracev1alpha1.OneAgentInstance{},
		}
		oneAgent := dynatracev1alpha1.OneAgent{Status: oneAgentStatus}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oneAgent},
		}

		res := nodesController.filterOneAgentFromList(oaList, "node_1")

		assert.Nil(t, res)
	}
	{
		oneAgentStatus := dynatracev1alpha1.OneAgentStatus{
			Items: map[string]dynatracev1alpha1.OneAgentInstance{
				"node_1": dynatracev1alpha1.OneAgentInstance{},
			},
		}
		oneAgent := dynatracev1alpha1.OneAgent{Status: oneAgentStatus}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oneAgent},
		}

		res := nodesController.filterOneAgentFromList(oaList, "node_1")

		assert.NotNil(t, res)
	}
	{
		oneAgentStatus := dynatracev1alpha1.OneAgentStatus{
			Items: map[string]dynatracev1alpha1.OneAgentInstance{
				"node_2": dynatracev1alpha1.OneAgentInstance{},
			},
		}
		oneAgent := dynatracev1alpha1.OneAgent{Status: oneAgentStatus}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oneAgent},
		}

		res := nodesController.filterOneAgentFromList(oaList, "node_1")

		assert.Nil(t, res)

	}
	{
		oneAgentStatus := dynatracev1alpha1.OneAgentStatus{
			Items: map[string]dynatracev1alpha1.OneAgentInstance{
				"node_1": dynatracev1alpha1.OneAgentInstance{},
				"node_2": dynatracev1alpha1.OneAgentInstance{},
			},
		}
		oneAgent := dynatracev1alpha1.OneAgent{Status: oneAgentStatus}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oneAgent},
		}

		res := nodesController.filterOneAgentFromList(oaList, "node_1")

		assert.NotNil(t, res)

	}
}

func TestNodesReconciler_NewSchedulableNode(t *testing.T) {
	nodeName := "new-node"
	fakeClient := fake.NewFakeClient(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName},
		})

	dtClient := &dtclient.MockDynatraceClient{}

	nodesController := NewController("dynatrace", fakeClient, staticDynatraceClient(dtClient))

	assert.NoError(t, nodesController.ReconcileNodes(nodeName))
	assert.Len(t, nodesController.nodeCordonedStatus, 1)
	assert.Contains(t, nodesController.nodeCordonedStatus, nodeName)
	assert.False(t, nodesController.nodeCordonedStatus[nodeName])
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestNodesReconciler_UnschedulableNode(t *testing.T) {
	nodeName := "unschedulable-node"
	fakeClient := fake.NewFakeClient(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Status: dynatracev1alpha1.OneAgentStatus{
				Items: map[string]dynatracev1alpha1.OneAgentInstance{
					nodeName: {IPAddress: "1.2.3.4"},
				},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetEntityIDForIP", "1.2.3.4").Return("HOST-42", nil)
	dtClient.On("SendEvent", mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)

	nodesController := NewController("dynatrace", fakeClient, staticDynatraceClient(dtClient))

	assert.NoError(t, nodesController.ReconcileNodes(nodeName))
	assert.Len(t, nodesController.nodeCordonedStatus, 1)
	assert.Contains(t, nodesController.nodeCordonedStatus, nodeName)
	assert.True(t, nodesController.nodeCordonedStatus[nodeName])
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestNodesReconciler_DeletedNode(t *testing.T) {
	nodeName := "deleted-node"
	fakeClient := fake.NewFakeClient(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Spec:       dynatracev1alpha1.OneAgentSpec{},
			Status: dynatracev1alpha1.OneAgentStatus{
				Items: map[string]dynatracev1alpha1.OneAgentInstance{
					nodeName: {IPAddress: "1.2.3.4"},
				},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetEntityIDForIP", "1.2.3.4").Return("HOST-42", nil)
	dtClient.On("SendEvent", mock.MatchedBy(func(e *dtclient.EventData) bool {
		return e.EventType == "MARKED_FOR_TERMINATION"
	})).Return(nil)

	nodesController := NewController("dynatrace", fakeClient, staticDynatraceClient(dtClient))

	assert.NoError(t, nodesController.ReconcileNodes(nodeName))
	assert.Len(t, nodesController.nodeCordonedStatus, 1)
	assert.Contains(t, nodesController.nodeCordonedStatus, nodeName)
	assert.True(t, nodesController.nodeCordonedStatus[nodeName])
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestNodesReconciler_UnschedulableNodeAndNoMatchingOneAgent(t *testing.T) {
	nodeName := "unschedulable-node"
	fakeClient := fake.NewFakeClient(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: nodeName},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.OneAgentSpec{
				NodeSelector: map[string]string{"a": "b"},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}

	nodesController := NewController("dynatrace", fakeClient, staticDynatraceClient(dtClient))

	assert.NoError(t, nodesController.ReconcileNodes(nodeName))
	assert.Empty(t, nodesController.nodeCordonedStatus)
	mock.AssertExpectationsForObjects(t, dtClient)
}

func staticDynatraceClient(c dtclient.Client) oneagent_utils.DynatraceClientFunc {
	return func(_ client.Client, oa *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {
		return c, nil
	}
}
