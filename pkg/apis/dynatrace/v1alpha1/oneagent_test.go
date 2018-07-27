package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOneAgent_BuildLabels(t *testing.T) {
	oa := newOneAgent()
	l := oa.BuildLabels()
	assert.Equal(t, l["dynatrace"], "oneagent")
	assert.Equal(t, l["oneagent"], "my-oneagent")
}

func TestOneAgent_BuildDaemonSet(t *testing.T) {
	oa := newOneAgent()
	ds := oa.BuildDaemonSet()
	assert.Equal(t, ds.APIVersion, "apps/v1")
	assert.Equal(t, ds.Kind, "DaemonSet")
	assert.Equal(t, ds.Name, oa.Name)
	assert.Equal(t, ds.Namespace, oa.Namespace)
}

func TestOneAgent_Validate(t *testing.T) {
	oa := newOneAgent()
	assert.Error(t, oa.Validate())
	oa.Spec.ApiUrl = "https://f.q.d.n/api"
	assert.NoError(t, oa.Validate())
}

func newOneAgent() *OneAgent {
	return &OneAgent{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OneAgent",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
	}
}

func newOneAgentSpec() *OneAgentSpec {
	return &OneAgentSpec{}
}
