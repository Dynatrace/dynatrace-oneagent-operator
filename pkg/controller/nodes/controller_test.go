package nodes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDetermineCustomResource(t *testing.T) {
	node := v1.Node{Spec: v1.NodeSpec{}}
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

func TestControllerGetSecret(t *testing.T) {
	{
		secret := &v1.Secret{StringData: map[string]string{}}
		secret.Name = "name"
		secret.Namespace = "namespace"
		nc := &Controller{}
		nc.kubernetesClient = fake.NewSimpleClientset(secret)

		s, err := nc.getSecret("name", "namespace")
		assert.Nil(t, s)
		assert.Error(t, err, "invalid secret name, missing token paasToken")
	}
	{
		secret := &v1.Secret{StringData: map[string]string{}}
		secret.Name = "name"
		secret.Namespace = "namespace"
		nc := &Controller{}
		nc.kubernetesClient = fake.NewSimpleClientset(secret)

		s, err := nc.getSecret("", "")
		assert.Nil(t, s)
		assert.Error(t, err, "invalid secret name")
	}
	{
		secret := &v1.Secret{
			Data: map[string][]byte{
				"paasToken": []byte("paasToken"),
				"apiToken":  []byte("apiToken"),
			}}
		secret.Name = "name"
		secret.Namespace = "namespace"
		nc := &Controller{}
		nc.kubernetesClient = fake.NewSimpleClientset(secret)

		s, err := nc.getSecret("name", "namespace")
		assert.NotNil(t, s)
		assert.NoError(t, err)
		assert.ObjectsAreEqualValues(secret, s)
	}
}
