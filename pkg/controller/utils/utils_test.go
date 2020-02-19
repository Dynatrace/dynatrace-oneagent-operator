package utils

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExtractToken(t *testing.T) {
	{
		secret := corev1.Secret{}
		_, err := extractToken(&secret, "test_token")
		assert.EqualError(t, err, "missing token test_token")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("")
		secret := corev1.Secret{Data: data}
		token, err := extractToken(&secret, "test_token")
		assert.NoError(t, err)
		assert.Equal(t, token, "")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("dynatrace_test_token")
		secret := corev1.Secret{Data: data}
		token, err := extractToken(&secret, "test_token")
		assert.NoError(t, err)
		assert.Equal(t, token, "dynatrace_test_token")
	}
	{
		data := map[string][]byte{}
		data["test_token"] = []byte("dynatrace_test_token \t \n")
		data["test_token_2"] = []byte("\t\n   dynatrace_test_token_2")
		secret := corev1.Secret{Data: data}
		token, err := extractToken(&secret, "test_token")
		token2, err := extractToken(&secret, "test_token_2")

		assert.NoError(t, err)
		assert.Equal(t, token, "dynatrace_test_token")
		assert.Equal(t, token2, "dynatrace_test_token_2")
	}
}

func TestVerifySecret(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{},
	}
	{
		err := verifySecret(secret)
		assert.Error(t, err)
	}
	{
		secret.Data[DynatraceApiToken] = []byte("DynatraceApiToken")
		err := verifySecret(secret)
		assert.Error(t, err)
	}
	{
		secret.Data[DynatracePaasToken] = []byte("DynatracePaasToken")
		err := verifySecret(secret)
		assert.NoError(t, err)
	}
}

func TestBuildDynatraceClient(t *testing.T) {
	namespace := "dynatrace"

	oa := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: namespace},
		Spec: dynatracev1alpha1.OneAgentSpec{
			ApiUrl: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: "custom-token",
		},
	}

	{
		fakeClient := fake.NewFakeClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-token", Namespace: namespace},
				Type:       corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"paasToken": []byte("42"),
					"apiToken":  []byte("43"),
				},
			},
		)

		_, err := BuildDynatraceClient(fakeClient, oa)
		assert.NoError(t, err)
	}

	{
		fakeClient := fake.NewFakeClient()
		_, err := BuildDynatraceClient(fakeClient, oa)
		assert.Error(t, err)
	}

	{
		fakeClient := fake.NewFakeClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "custom-token", Namespace: namespace},
				Type:       corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"paasToken": []byte("42"),
				},
			},
		)
		_, err := BuildDynatraceClient(fakeClient, oa)
		assert.Error(t, err)
	}
}
