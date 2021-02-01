package oneagenttests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func prepareDefaultEnvironment(t *testing.T) (string, client.Client) {
	apiURL := os.Getenv(keyApiURL)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

	clt := e2e.CreateClient(t)
	assert.NotNil(t, clt)

	err := e2e.PrepareEnvironment(clt, namespace)
	require.NoError(t, err)

	return apiURL, clt
}

func createMinimumViableOneAgent(apiURL string) v1alpha1.OneAgent {
	return v1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      testName,
		},
		Spec: v1alpha1.OneAgentSpec{
			BaseOneAgentSpec: v1alpha1.BaseOneAgentSpec{
				APIURL: apiURL,
				Tokens: e2e.TokenSecretName,
			},
			Image: testImage,
		}}
}

func deployOneAgent(t *testing.T, clt client.Client, oneAgent *v1alpha1.OneAgent) e2e.PhaseWait {
	err := clt.Create(context.TODO(), oneAgent)
	assert.NoError(t, err)

	phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
	err = phaseWait.WaitForPhase(v1alpha1.Deploying)
	assert.NoError(t, err)

	return phaseWait
}

func findOneAgentPods(t *testing.T, clt client.Client) (*v1alpha1.OneAgent, *corev1.PodList) {
	instance := &v1alpha1.OneAgent{}
	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: testName}, instance)
	assert.NoError(t, err)

	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(buildLabels(instance.Name)),
	}
	err = clt.List(context.TODO(), podList, listOps...)
	assert.NoError(t, err)

	return instance, podList
}

func buildLabels(name string) map[string]string {
	return map[string]string{
		"dynatrace": "oneagent",
		"oneagent":  name,
	}
}
