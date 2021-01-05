package oneagenttests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/e2e"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// Imports auth providers. see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace     = "dynatrace"
	testName      = "test-name"
	testImage     = "test-image:latest"
	keyApiURL     = "DYNATRACE_API_URL"
	maxWaitCycles = 5
)

func TestImmutableImage(t *testing.T) {
	t.Run(`pull secret is created if image is unset`, func(t *testing.T) {
		apiURL := os.Getenv(keyApiURL)
		assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

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
					APIURL:            apiURL,
					Tokens:            e2e.TokenSecretName,
					UseImmutableImage: true,
				}}}
		err = clt.Create(context.TODO(), &oneAgent)
		assert.NoError(t, err)

		phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
		err = phaseWait.WaitForPhase(v1alpha1.Deploying)
		assert.NoError(t, err)

		pullSecret := v1.Secret{}
		err = clt.Get(context.TODO(), client.ObjectKey{Name: buildPullSecretName(), Namespace: namespace}, &pullSecret)
		assert.NoError(t, err)
	})
	t.Run(`no pull secret exists if image is set`, func(t *testing.T) {
		apiURL := os.Getenv(keyApiURL)
		assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

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
					APIURL:            apiURL,
					Tokens:            e2e.TokenSecretName,
					UseImmutableImage: true,
				},
				Image: testImage,
			}}
		err = clt.Create(context.TODO(), &oneAgent)
		assert.NoError(t, err)

		phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
		err = phaseWait.WaitForPhase(v1alpha1.Deploying)
		assert.NoError(t, err)

		pullSecret := v1.Secret{}
		err = clt.Get(context.TODO(), client.ObjectKey{Name: buildPullSecretName(), Namespace: namespace}, &pullSecret)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func buildPullSecretName() string {
	return fmt.Sprintf("%s-pull-secret", testName)
}
