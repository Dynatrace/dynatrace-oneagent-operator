package nodes

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testNodeController(t *testing.T) *Controller {

	spec := dynatracev1alpha1.OneAgentSpec{}
	dynatracev1alpha1.SetDefaults_OneAgentSpec(&spec)
	oa := &dynatracev1alpha1.OneAgent{Spec: spec}

	node := &v1.Node{Spec: v1.NodeSpec{}}
	node.Name = "node_1"
	node.Labels = map[string]string{"test_node": "test_label"}

	nc := &Controller{}
	nc.kubernetesClient = fake.NewSimpleClientset(oa, node)
	return nc
}

func TestControllerIsSubset(t *testing.T) {

	nc := &Controller{}
	{
		child := make(map[string]string)
		parent := make(map[string]string)
		res := nc.isSubset(child, parent)
		assert.True(t, res, "length is zero for both")
	}
	{
		child := make(map[string]string)
		parent := make(map[string]string, 2)
		res := nc.isSubset(child, parent)
		assert.True(t, res, "parent > child, but map nil")
	}
	{
		child := make(map[string]string, 2)
		parent := make(map[string]string)
		res := nc.isSubset(child, parent)
		assert.True(t, res, "child > parent, but maps are nil")
	}
	{
		child := map[string]string{"A": "a", "B": "b", "C": "c"}
		parent := make(map[string]string, 2)
		res := nc.isSubset(child, parent)
		assert.False(t, res, "child > parent, but parent is nil")
	}
	{
		child := map[string]string{"A": "a", "B": "b", "C": "c"}
		parent := map[string]string{"A": "a", "B": "b", "C": "c"}
		res := nc.isSubset(child, parent)
		assert.True(t, res, "child == parent, maps eq in values")
	}
	{
		child := map[string]string{"A": "a", "B": "b"}
		parent := map[string]string{"A": "a", "B": "b", "C": "c"}
		res := nc.isSubset(child, parent)
		assert.True(t, res, "child < parent, but maps are not nil")
	}
	{
		child := map[string]string{"A": "a", "B": "b", "C": "c"}
		parent := map[string]string{"A": "1", "B": "2", "C": "3"}
		res := nc.isSubset(child, parent)
		assert.False(t, res, "child == parent, but maps are not equal in vals")
	}
	{
		child := map[string]string{"A": "a", "B": "b", "C": "c"}
		parent := map[string]string{"A": "1", "B": "b"}
		res := nc.isSubset(child, parent)
		assert.False(t, res, "child >= parent, but only one label matches")
	}
}

func TestDetermineCustomResource(t *testing.T) {

	node := &v1.Node{Spec: v1.NodeSpec{}}
	node.Name = "node_1"
	node.Labels = map[string]string{
		"test_node": "test_label", "beta.kubernetes.io/os": "linux"}

	nc := &Controller{}
	{
		spec := dynatracev1alpha1.OneAgentSpec{}
		dynatracev1alpha1.SetDefaults_OneAgentSpec(&spec)
		spec.NodeSelector["test_node"] = "test_label"
		oa := dynatracev1alpha1.OneAgent{Spec: spec}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oa},
		}

		res := nc.determineCustomResource(oaList, node)
		assert.NotNil(t, res, "result is found")
		assert.ObjectsAreEqualValues(res, oa)
	}
	{
		spec := dynatracev1alpha1.OneAgentSpec{}
		dynatracev1alpha1.SetDefaults_OneAgentSpec(&spec)
		oa := dynatracev1alpha1.OneAgent{Spec: spec}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oa},
		}

		res := nc.determineCustomResource(oaList, node)
		assert.NotNil(t, res, "result is found")
		assert.ObjectsAreEqualValues(res, oa)
	}
	{
		spec := dynatracev1alpha1.OneAgentSpec{}
		dynatracev1alpha1.SetDefaults_OneAgentSpec(&spec)
		spec.NodeSelector["test_node"] = "test_label_different"
		oa := dynatracev1alpha1.OneAgent{Spec: spec}
		oaList := &dynatracev1alpha1.OneAgentList{
			Items: []dynatracev1alpha1.OneAgent{oa},
		}

		res := nc.determineCustomResource(oaList, node)
		assert.Nil(t, res, "result is not found")
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

func TestControllerGetInternalIPForNode(t *testing.T) {
	node := &v1.Node{Spec: v1.NodeSpec{}}
	node.Name = "node_1"
	node.Labels = map[string]string{
		"test_node": "test_label", "beta.kubernetes.io/os": "linux"}

	nc := &Controller{}
	{
		ip := nc.getInternalIPForNode(node)
		assert.Empty(t, ip)
	}
	{
		node.Status.Addresses = []v1.NodeAddress{v1.NodeAddress{
			Type: v1.NodeInternalIP, Address: "some.address.com"}}
		ip := nc.getInternalIPForNode(node)
		assert.NotEmpty(t, ip)
		assert.Equal(t, ip, "some.address.com")
	}
	{
		node.Status.Addresses = []v1.NodeAddress{v1.NodeAddress{
			Type: v1.NodeExternalDNS, Address: "some.address.com"}}
		ip := nc.getInternalIPForNode(node)
		assert.Empty(t, ip)
	}
}
