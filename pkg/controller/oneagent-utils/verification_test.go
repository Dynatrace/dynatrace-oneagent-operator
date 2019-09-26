package oneagent_utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestExtractToken(t *testing.T) {
	secret := &v1.Secret{
		Data: map[string][]byte{},
	}
	{
		val, err := ExtractToken(secret, "key")
		assert.Error(t, err)
		assert.Empty(t, val)
	}
	{
		secret.Data["key"] = []byte("val")
		val, err := ExtractToken(secret, "key")
		assert.NoError(t, err)
		assert.Equal(t, val, "val")
	}
}

func TestVerifyToken(t *testing.T) {
	secret := &v1.Secret{
		Data: map[string][]byte{},
	}
	{
		err := VerifySecret(secret)
		assert.Error(t, err)
	}
	{
		secret.Data[DynatraceApiToken] = []byte("DynatraceApiToken")
		err := VerifySecret(secret)
		assert.Error(t, err)
	}
	{
		secret.Data[DynatracePaasToken] = []byte("DynatracePaasToken")
		err := VerifySecret(secret)
		assert.NoError(t, err)
	}
}
