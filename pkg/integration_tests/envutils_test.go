package oneagent

// This file includes utilities to start an environment with API Server, and a configured reconciler.

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	DefaultTestAPIURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	DefaultTestNamespace = "dynatrace"
)

var testEnvironmentCRDs = []*apiextensionsv1beta1.CustomResourceDefinition{
	&apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oneagents.dynatrace.com",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "dynatrace.com",
			Version: "v1alpha1",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "OneAgent",
				ListKind: "OneAgentList",
				Plural:   "oneagents",
				Singular: "oneagent",
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	},
	&apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virtualservices.networking.istio.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "VirtualService",
				ListKind: "VirtualServiceList",
				Plural:   "virtualservices",
				Singular: "virtualservice",
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	},
	&apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "serviceentries.networking.istio.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "ServiceEntry",
				ListKind: "ServiceEntryList",
				Plural:   "serviceentries",
				Singular: "serviceentry",
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
			Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	},
}

func init() {
	// Register OneAgent and Istio object schemas.
	apis.AddToScheme(scheme.Scheme)
}

type ControllerTestEnvironment struct {
	CommunicationHosts []string
	Client             client.Client
	Reconciler         *oneagent.ReconcileOneAgent

	server *envtest.Environment
}

func newTestEnvironment() (*ControllerTestEnvironment, error) {
	kubernetesAPIServer := &envtest.Environment{
		KubeAPIServerFlags: append(envtest.DefaultKubeAPIServerFlags, "--allow-privileged"),
		CRDs:               testEnvironmentCRDs,
	}

	// TODO: we shouldn't need to set environment variables. Remove usages from our code.
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")

	cfg, err := kubernetesAPIServer.Start()
	if err != nil {
		return nil, err
	}

	kubernetesClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		kubernetesAPIServer.Stop()
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), buildDynatraceClientSecret()); err != nil {
		kubernetesAPIServer.Stop()
		return nil, err
	}

	communicationHosts := []string{
		"https://endpoint1.test.com/communication",
		"https://endpoint2.test.com/communication",
	}
	environment := &ControllerTestEnvironment{
		server:             kubernetesAPIServer,
		Client:             kubernetesClient,
		CommunicationHosts: communicationHosts,
		Reconciler: oneagent.NewOneAgentReconciler(kubernetesClient, scheme.Scheme, cfg,
			logf.ZapLoggerTo(os.Stdout, true), mockDynatraceClientFunc(communicationHosts)),
	}

	return environment, nil
}

func (e *ControllerTestEnvironment) Stop() error {
	return e.server.Stop()
}

func (e *ControllerTestEnvironment) AddOneAgent(n string, s *dynatracev1alpha1.OneAgentSpec) error {
	dynatracev1alpha1.SetDefaults_OneAgentSpec(s)

	return e.Client.Create(context.TODO(), &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: DefaultTestNamespace,
		},
		Spec: *s,
	})
}

func newReconciliationRequest(oaName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      oaName,
			Namespace: DefaultTestNamespace,
		},
	}
}

func mockDynatraceClientFunc(communicationHosts []string) oneagent.DynatraceClientFunc {
	return func(oa *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {
		commHosts := make([]dtclient.CommunicationHost, len(communicationHosts))
		for i, c := range communicationHosts {
			commHosts[i] = dtclient.CommunicationHost{Protocol: "https", Host: c, Port: 443}
		}

		dtc := new(dtclient.MockDynatraceClient)
		dtc.On("GetLatestAgentVersion").Return("17")
		dtc.On("GetVersionForIp", "127.0.0.1").Return("1.2.3", nil)
		dtc.On("GetCommunicationHosts").Return(commHosts, nil)
		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: "https",
			Host:     DefaultTestAPIURL,
			Port:     443,
		}, nil)

		return dtc, nil
	}
}

func buildDynatraceClientSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-test",
			Namespace: DefaultTestNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"paasToken": []byte("42"),
			"apiToken":  []byte("43"),
		},
	}
}
