package oneagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func InitMockServer(t *testing.T) *httptest.Server {
	list := &metav1.APIGroupList{
		Groups: []metav1.APIGroup{
			{
				Name: "networking.istio.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{GroupVersion: "v1alpha3", Version: "v1alpha3"},
					{GroupVersion: "v1alpha3", Version: "v1alpha3"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var resources interface{}
		switch req.URL.Path {
		case "/apis":
			resources = list
		default:
			// t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(resources)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))

	return server
}

func TestReconcileOneAgent_ReconcileIstio(t *testing.T) {

	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")

	server := InitMockServer(t)
	defer server.Close()

	var (
		name      = "dynatrace-oneagent"
		namespace = "dynatrace"
	)

	oa := newOneAgentSpec()
	oa.ApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	oa.Tokens = "token_test"
	oa.EnableIstio = true
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)

	instance := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *oa,
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token_test",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"paasToken": []byte("42"),
			"apiToken":  []byte("43"),
		},
	}

	scheme := scheme.Scheme
	scheme.AddKnownTypes(dynatracev1alpha1.SchemeGroupVersion, instance)

	client := fake.NewFakeClient(instance)
	client.Create(context.TODO(), secret)

	cfg := &restclient.Config{Host: server.URL}

	// reconcile oneagent
	reconcileOA := &ReconcileOneAgent{client: client, scheme: scheme, config: cfg}

	// mocking the request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	_, err := reconcileOA.Reconcile(req)
	if err != nil {
		t.Fatalf("error reconciling: %v", err)
	}
	virtualService := getGVK(client, istio.VirtualServiceGVK)
	if virtualService == nil {
		t.Error("no istio virtual services objects formed")
	}
	serviceEntry := getGVK(client, istio.ServiceEntryGVK)
	if serviceEntry == nil {
		t.Error("no istio objects for service entry")
	}
}

func getGVK(fake client.Client, gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	listOpts := &client.ListOptions{
		Namespace: "dynatrace",
	}

	fake.List(context.TODO(), listOpts, list)
	return list
}
