package utils

import (
	"context"
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DynatracePaasToken = "paasToken"
	DynatraceApiToken  = "apiToken"
)

var logger = log.Log.WithName("dynatrace.utils")

// DynatraceClientFunc defines handler func for dynatrace client
type DynatraceClientFunc func(rtc client.Client, instance dynatracev1alpha1.OneAgentInterface) (dtclient.Client, error)

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(rtc client.Client, instance dynatracev1alpha1.OneAgentInterface) (dtclient.Client, error) {
	ns := instance.GetNamespace()
	spec := instance.GetSpec()

	secret := &corev1.Secret{}
	err := rtc.Get(context.TODO(), client.ObjectKey{Name: GetTokensName(instance), Namespace: ns}, secret)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if err = verifySecret(secret); err != nil {
		return nil, err
	}

	// initialize dynatrace client
	var opts []dtclient.Option
	if spec.SkipCertCheck {
		opts = append(opts, dtclient.SkipCertificateValidation(true))
	}

	if p := spec.Proxy; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := rtc.Get(context.TODO(), client.ObjectKey{Name: p.ValueFrom, Namespace: ns}, proxySecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get proxy secret: %w", err)
			}

			proxyURL, err := extractToken(proxySecret, "proxy")
			if err != nil {
				return nil, fmt.Errorf("failed to extract proxy secret field: %w", err)
			}
			opts = append(opts, dtclient.Proxy(proxyURL))
		} else if p.Value != "" {
			opts = append(opts, dtclient.Proxy(p.Value))
		}
	}

	if spec.TrustedCAs != "" {
		certs := &corev1.ConfigMap{}
		if err := rtc.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: spec.TrustedCAs}, certs); err != nil {
			return nil, fmt.Errorf("failed to get certificate configmap: %w", err)
		}
		if certs.Data["certs"] == "" {
			return nil, fmt.Errorf("failed to extract certificate configmap field: missing field certs")
		}
		opts = append(opts, dtclient.Certs([]byte(certs.Data["certs"])))
	}

	apiToken, err := extractToken(secret, DynatraceApiToken)
	if err != nil {
		return nil, err
	}

	paasToken, err := extractToken(secret, DynatracePaasToken)
	if err != nil {
		return nil, err
	}

	return dtclient.NewClient(spec.APIURL, apiToken, paasToken, opts...)
}

func extractToken(secret *v1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func verifySecret(secret *v1.Secret) error {
	for _, token := range []string{DynatracePaasToken, DynatraceApiToken} {
		_, err := extractToken(secret, token)
		if err != nil {
			return fmt.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}

// StaticDynatraceClient creates a DynatraceClientFunc always returning c.
func StaticDynatraceClient(c dtclient.Client) DynatraceClientFunc {
	return func(_ client.Client, oa dynatracev1alpha1.OneAgentInterface) (dtclient.Client, error) {
		return c, nil
	}
}

func GetTokensName(obj dynatracev1alpha1.BaseOneAgent) string {
	if tkns := obj.GetSpec().Tokens; tkns != "" {
		return tkns
	}
	return obj.GetName()
}

// GetDeployment returns the Deployment object who is the owner of this pod.
func GetDeployment(c client.Client, ns string) (*appsv1.Deployment, error) {
	pod, err := k8sutil.GetPod(context.TODO(), c, ns)
	if err != nil {
		return nil, err
	}

	rsOwner := metav1.GetControllerOf(pod)
	if rsOwner == nil {
		return nil, fmt.Errorf("no controller found for Pod: %s", pod.Name)
	} else if rsOwner.Kind != "ReplicaSet" {
		return nil, fmt.Errorf("unexpected controller found for Pod: %s, kind: %s", pod.Name, rsOwner.Kind)
	}

	var rs appsv1.ReplicaSet
	if err := c.Get(context.TODO(), client.ObjectKey{Name: rsOwner.Name, Namespace: ns}, &rs); err != nil {
		return nil, err
	}

	dOwner := metav1.GetControllerOf(&rs)
	if dOwner == nil {
		return nil, fmt.Errorf("no controller found for ReplicaSet: %s", pod.Name)
	} else if dOwner.Kind != "Deployment" {
		return nil, fmt.Errorf("unexpected controller found for ReplicaSet: %s, kind: %s", pod.Name, dOwner.Kind)
	}

	var d appsv1.Deployment
	if err := c.Get(context.TODO(), client.ObjectKey{Name: dOwner.Name, Namespace: ns}, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
