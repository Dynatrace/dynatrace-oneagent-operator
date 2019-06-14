package oneagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func mockBuildDynatraceClient(instance *dynatracev1alpha1.OneAgent) (dtclient.Client, error) {

	commHosts := []dtclient.CommunicationHost{
		dtclient.CommunicationHost{
			Protocol: "https",
			Host:     "https://endpoint1.dev.ruxitlabs.com/communication",
			Port:     443,
		},
		dtclient.CommunicationHost{
			Protocol: "https",
			Host:     "https://endpoint2.dev.ruxitlabs.com/communication",
			Port:     443,
		},
	}

	dtc := new(MyDynatraceClient)
	dtc.On("GetVersionForIp", "127.0.0.1").Return("1.2.3", nil)
	dtc.On("GetCommunicationHosts").Return(commHosts, nil)
	dtc.On("GetAPIURLHost").Return(dtclient.CommunicationHost{
		Protocol: "https",
		Host:     testAPIUrl,
		Port:     443,
	}, nil)

	return dtc, nil
}

func initMockServer(t *testing.T) *httptest.Server {
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

func setupReconciler(t *testing.T) (*ReconcileOneAgent, client.Client) {
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")

	server := initMockServer(t)
	defer server.Close()

	oa := newOneAgentSpec()
	oa.ApiUrl = testAPIUrl
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
	reconcileOA.dynatraceClientFunc = mockBuildDynatraceClient

	return reconcileOA, client
}

func TestReconcileOneAgent_ReconcileOnEmptyEnvironment(t *testing.T) {
	reconcileOA, client := setupReconciler(t)

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

	// Check if deamonset has been created and has correct namespace and name.
	dsActual := &appsv1.DaemonSet{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, dsActual)
	if err != nil {
		t.Fatalf("get deamonset: (%v)", err)
	}
	if dsActual.Namespace != namespace {
		t.Errorf("wrong namespace, expected %v, got %v", namespace, dsActual.Namespace)
	}
	if dsActual.GetObjectMeta().GetName() != name {
		t.Errorf("wrong name, expected %v, got %v", name, dsActual.GetObjectMeta().GetName())
	}
}
