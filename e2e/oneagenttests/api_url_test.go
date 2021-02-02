// +build e2e

package oneagenttests

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
	"github.com/Dynatrace/dynatrace-oneagent-operator/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// Imports auth providers. see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const keyEnvironmentId = "DYNATRACE_ENVIRONMENT_ID"

func TestApiURL(t *testing.T) {
	apiURL := os.Getenv(keyApiURL)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

	environmentId := os.Getenv(keyEnvironmentId)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyEnvironmentId))

	clt := e2e.CreateClient(t)
	assert.NotNil(t, clt)

	err := e2e.PrepareEnvironment(clt, namespace)
	require.NoError(t, err)

	oneAgent := v1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      testName,
		},
		Spec: v1alpha1.OneAgentSpec{
			BaseOneAgentSpec: v1alpha1.BaseOneAgentSpec{
				APIURL: apiURL,
				Tokens: e2e.TokenSecretName,
			}}}
	err = clt.Create(context.TODO(), &oneAgent)
	assert.NoError(t, err)

	phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
	err = phaseWait.WaitForPhase(v1alpha1.Deploying)
	assert.NoError(t, err)

	err = phaseWait.WaitForPhase(v1alpha1.Running)
	assert.NoError(t, err)

	apiToken, paasToken := e2e.GetTokensFromEnv()
	dtc, err := dtclient.NewClient(apiURL, apiToken, paasToken)
	assert.NoError(t, err)

	connectionInfo, err := dtc.GetConnectionInfo()
	assert.NoError(t, err)
	assert.NotNil(t, connectionInfo)
	assert.Equal(t, environmentId, connectionInfo.TenantUUID)
	assert.True(t, containsAPIConnectionHost(connectionInfo, apiURL))

	apiScopes, err := dtc.GetTokenScopes(apiToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, apiScopes)

	paasScopes, err := dtc.GetTokenScopes(paasToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, paasScopes)
}

func containsAPIConnectionHost(connectionInfo dtclient.ConnectionInfo, apiURL string) bool {
	apiUrl, err := url.Parse(apiURL)
	if err != nil {
		return false
	}

	for _, connectionHost := range connectionInfo.CommunicationHosts {
		if connectionHost.Host == apiUrl.Host {
			return true
		}
	}
	return false
}
