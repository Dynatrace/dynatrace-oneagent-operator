package oneagent_utils

import (
	"fmt"
	"k8s.io/api/core/v1"
	"strings"
)

const (
	DynatracePaasToken = "paasToken"
	DynatraceApiToken  = "apiToken"
)

func ExtractToken(secret *v1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func VerifySecret(secret *v1.Secret) error {
	var err error

	for _, token := range []string{DynatracePaasToken, DynatraceApiToken} {
		_, err = ExtractToken(secret, token)
		if err != nil {
			return fmt.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}
