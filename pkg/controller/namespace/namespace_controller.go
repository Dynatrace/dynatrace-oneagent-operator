package namespace

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"text/template"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/webhook"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Add(mgr manager.Manager) error {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	return add(mgr, &ReconcileNamespaces{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		namespace: ns,
		logger:    log.Log.WithName("namespaces.controller"),
	})
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Namespaces
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

type ReconcileNamespaces struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func (r *ReconcileNamespaces) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name

	ctx := context.TODO()

	log := r.logger.WithValues("name", targetNS)
	log.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query Namespace: %w", err)
	}

	if ns.Labels == nil {
		return reconcile.Result{}, nil
	}

	oaName := ns.Labels[webhook.LabelInstance]
	if oaName == "" {
		return reconcile.Result{}, nil
	}

	script, err := newScript(ctx, r.client, oaName, r.namespace)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to generate init script: %w", err)
	}

	data, err := script.generate()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to generate script: %w", err)
	}

	var cfg corev1.Secret

	// The default cache-based Client doesn't support cross-namespace queries, unless configured to do so in Manager
	// Options. However, this is our only use-case for it, so using the non-cached Client instead.

	err = r.apiReader.Get(ctx, client.ObjectKey{Name: webhook.SecretConfigName, Namespace: targetNS}, &cfg)
	if errors.IsNotFound(err) {
		log.Info("Creating OneAgent config secret")
		if err := r.client.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.SecretConfigName,
				Namespace: targetNS,
			},
			Data: data,
		}); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to create config Secret: %w", err)
		}
		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query for config Secret: %w", err)
	}

	if !reflect.DeepEqual(data, cfg.Data) {
		log.Info("Updating OneAgent config secret")
		cfg.Data = data
		if err := r.client.Update(ctx, &cfg); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update config Secret: %w", err)
		}
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

type script struct {
	OneAgent   *dynatracev1alpha1.OneAgent
	PaaSToken  string
	Proxy      string
	TrustedCAs []byte
}

func newScript(ctx context.Context, c client.Client, oaName, ns string) (*script, error) {
	var oa dynatracev1alpha1.OneAgent
	if err := c.Get(ctx, client.ObjectKey{Name: oaName, Namespace: ns}, &oa); err != nil {
		return nil, fmt.Errorf("failed to query OneAgent: %w", err)
	}

	var tkns corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(&oa), Namespace: ns}, &tkns); err != nil {
		return nil, fmt.Errorf("failed to query tokens: %w", err)
	}

	var proxy string
	if oa.Spec.Proxy != nil {
		if oa.Spec.Proxy.ValueFrom != "" {
			var ps corev1.Secret
			if err := c.Get(ctx, client.ObjectKey{Name: oa.Spec.Proxy.ValueFrom, Namespace: ns}, &ps); err != nil {
				return nil, fmt.Errorf("failed to query proxy: %w", err)
			}
			proxy = string(ps.Data["proxy"])
		} else if oa.Spec.Proxy.Value != "" {
			proxy = oa.Spec.Proxy.Value
		}
	}

	var trustedCAs []byte
	if oa.Spec.TrustedCAs != "" {
		var cam corev1.ConfigMap
		if err := c.Get(ctx, client.ObjectKey{Name: oa.Spec.TrustedCAs, Namespace: ns}, &cam); err != nil {
			return nil, fmt.Errorf("failed to query ca: %w", err)
		}
		trustedCAs = []byte(cam.Data["proxy"])
	}

	return &script{
		OneAgent:   &oa,
		PaaSToken:  string(tkns.Data[utils.DynatracePaasToken]),
		Proxy:      proxy,
		TrustedCAs: trustedCAs,
	}, nil
}

var scriptTmpl = template.Must(template.New("initScript").Parse(`#!/usr/bin/env bash

set -eu

api_url="{{.OneAgent.Spec.ApiUrl}}"
config_dir="/mnt/config"
paas_token="{{.PaaSToken}}"
proxy="{{.Proxy}}"
skip_cert_checks="{{if .OneAgent.Spec.SkipCertCheck}}true{{else}}false{{end}}"
custom_ca="{{if .TrustedCAs}}true{{else}}false{{end}}"

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
`))

func (s *script) generate() (map[string][]byte, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, s); err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"init.sh": buf.Bytes(),
	}

	if s.TrustedCAs != nil {
		data["ca.pem"] = s.TrustedCAs
	}

	if s.Proxy != "" {
		data["proxy"] = []byte(s.Proxy)
	}

	return data, nil
}
