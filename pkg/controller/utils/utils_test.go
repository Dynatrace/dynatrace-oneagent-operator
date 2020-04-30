package utils

import (
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
			BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
				ApiUrl: "https://ENVIRONMENTID.live.dynatrace.com/api",
				Tokens: "custom-token",
			},
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

// GetDeployment returns the Deployment object who is the owner of this pod.
func TestGetDeployment(t *testing.T) {
	const ns = "dynatrace"

	os.Setenv(k8sutil.PodNameEnvVar, "mypod")
	trueVar := true

	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "myreplicaset", Controller: &trueVar},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myreplicaset",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "mydeployment", Controller: &trueVar},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydeployment",
				Namespace: ns,
			},
		})

	deploy, err := GetDeployment(fakeClient, "dynatrace")
	require.NoError(t, err)
	assert.Equal(t, "mydeployment", deploy.Name)
	assert.Equal(t, "dynatrace", deploy.Namespace)
}
