package namespace

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func init() {
	apis.AddToScheme(scheme.Scheme)
}

func TestPodInjection(t *testing.T) {
	c := fake.NewFakeClient(
		&dynatracev1alpha1.OneAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.OneAgentSpec{
				ApiUrl: "https://test-url/api",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace",
				Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
		},
	)

	r := ReconcileNamespaces{
		client:    c,
		apiReader: c,
		logger:    logf.ZapLoggerTo(os.Stdout, true),
		namespace: "dynatrace",
	}

	_, err := r.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
	assert.NoError(t, err)

	var nsSecret corev1.Secret
	require.NoError(t, c.Get(context.TODO(), client.ObjectKey{
		Name:      "dynatrace-oneagent-config",
		Namespace: "test-namespace",
	}, &nsSecret))

	require.Equal(t, map[string][]byte{
		"init.sh": []byte(`#!/usr/bin/env bash

set -eu

api_url="https://test-url/api"
config_dir="/mnt/config"
paas_token="42"
proxy=""
skip_cert_checks="false"
custom_ca="false"

archive=$(mktemp)

curl_params=(
	"--silent"
	"--output" "${archive}"
	"--header" "Authorization: Api-Token ${paas_token}"
	"${api_url}/v1/deployment/installer/agent/unix/paas/latest?flavor=${FLAVOR}&include=${TECHNOLOGIES}"
)

if [[ "${skip_cert_checks}" == "true" ]]; then
	curl_params+=("--insecure")
fi

if [[ "${custom_ca}" == "true" ]]; then
	curl_params+=("--cacert" "${config_dir}/ca.pem")
fi

if [[ "${proxy}" != "" ]]; then
	curl_params+=("--proxy" "${proxy}")
fi

echo "Downloading OneAgent package..."
curl "${curl_params[@]}"

echo "Unpacking OneAgent package..."
unzip -o -d /opt/dynatrace/oneagent "${archive}"
rm -f "${archive}"

echo "Configuring OneAgent..."
mkdir -p /opt/dynatrace/oneagent/agent/conf/pod
mkdir -p /opt/dynatrace/oneagent/agent/conf/node

echo -n "/opt/dynatrace/oneagent/agent/lib64/liboneagentproc.so" >> /opt/dynatrace/oneagent/ld.so.preload
echo -n "${NODENAME}" > /opt/dynatrace/oneagent/agent/conf/node/name
echo -n "${NODEIP}" > /opt/dynatrace/oneagent/agent/conf/node/ip
`),
	}, nsSecret.Data)
}
