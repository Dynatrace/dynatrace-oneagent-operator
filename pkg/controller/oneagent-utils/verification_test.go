package oneagent_utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
