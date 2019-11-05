package oneagent_utils

import (
	"context"
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DynatracePaasToken = "paasToken"
	DynatraceApiToken  = "apiToken"
)

// DynatraceClientFunc defines handler func for dynatrace client
type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.OneAgent) (dtclient.Client, error)

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(rtc client.Client, instance *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {
	secret := &corev1.Secret{}
	err := rtc.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: instance.Name}, secret)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	if err = VerifySecret(secret); err != nil {
		return nil, err
	}

	// initialize dynatrace client
	var certificateValidation = dtclient.SkipCertificateValidation(instance.Spec.SkipCertCheck)
	apiToken, _ := ExtractToken(secret, DynatraceApiToken)
	paasToken, _ := ExtractToken(secret, DynatracePaasToken)
	dtc, err := dtclient.NewClient(instance.Spec.ApiUrl, apiToken, paasToken, certificateValidation)

	return dtc, err
}

func ExtractToken(secret *v1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func VerifySecret(secret *v1.Secret) error {
	for _, token := range []string{DynatracePaasToken, DynatraceApiToken} {
		_, err := ExtractToken(secret, token)
		if err != nil {
			return fmt.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}
