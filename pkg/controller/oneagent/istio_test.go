package oneagent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	fakeistio "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned/fake"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIstioClient_CreateIstioObjects(t *testing.T) {

	buffer := bytes.NewBufferString("{\"apiVersion\":\"networking.istio.io/v1alpha3\",\"kind\":\"VirtualService\",\"metadata\":{\"clusterName\":\"\",\"creationTimestamp\":\"2018-11-26T03:19:57Z\",\"generation\":1,\"name\":\"test-virtual-service\",\"namespace\":\"istio-system\",\"resourceVersion\":\"1297970\",\"selfLink\":\"/apis/networking.istio.io/v1alpha3/namespaces/istio-system/virtualservices/test-virtual-service\",\"uid\":\"266fdacc-f12a-11e8-9e1d-42010a8000ff\"},\"spec\":{\"gateways\":[\"test-gateway\"],\"hosts\":[\"*\"],\"http\":[{\"match\":[{\"uri\":{\"prefix\":\"/\"}}],\"route\":[{\"destination\":{\"host\":\"test-service\",\"port\":{\"number\":8080}}}],\"timeout\":\"10s\"}]}}\n")

	vs := istiov1alpha3.VirtualService{}
	err := json.Unmarshal(buffer.Bytes(), &vs)

	ic := fakeistio.NewSimpleClientset(&vs)

	vsList, err := ic.NetworkingV1alpha3().VirtualServices("istio-system").List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", DefaultTestNamespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestIstioClient_BuildDynatraceVirtualService(t *testing.T) {
	os.Setenv(k8sutil.WatchNamespaceEnvVar, DefaultTestNamespace)

	buffer := istio.BuildVirtualService("dt-vs", "ENVIRONMENTID.live.dynatrace.com", 443, "https")
	vs := istiov1alpha3.VirtualService{}
	err := json.Unmarshal(buffer, &vs)
	if err != nil {
		t.Errorf("Failed to marshal json %s", err)
	}
	ic := fakeistio.NewSimpleClientset(&vs)
	vsList, err := ic.NetworkingV1alpha3().VirtualServices(DefaultTestNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", DefaultTestNamespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestReconcileOneAgent_ReconcileIstio(t *testing.T) {
	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	e.AddOneAgent("oneagent", &dynatracev1alpha1.OneAgentSpec{
		ApiUrl:      DefaultTestAPIURL,
		Tokens:      "token-test",
		EnableIstio: true,
	})

	req := newReconciliationRequest("oneagent")

	// For the first reconciliation, we only create Istio objects for the API URL.
	_, err = e.Reconciler.Reconcile(req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 1, 1)

	// Once the API URL is open, we create Istio objects for each communication endpoint.
	_, err = e.Reconciler.Reconcile(req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 3, 3)

	// Add a new communication endpoint.
	e.CommunicationHosts = append(e.CommunicationHosts, "https://endpoint3.dev.ruxitlabs.com/communication")
	_, err = e.Reconciler.Reconcile(req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 4, 4)

	// Remove two communication endpoints.
	e.CommunicationHosts = e.CommunicationHosts[2:]
	_, err = e.Reconciler.Reconcile(req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 2, 2)
}

func TestReconcileOneAgent_ReconcileIstioWithMultipleOneAgentObjects(t *testing.T) {
	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	e.AddOneAgent("oneagent1", &dynatracev1alpha1.OneAgentSpec{
		ApiUrl:      DefaultTestAPIURL,
		Tokens:      "token-test",
		EnableIstio: true,
	})

	e.AddOneAgent("oneagent2", &dynatracev1alpha1.OneAgentSpec{
		ApiUrl:      DefaultTestAPIURL,
		Tokens:      "token-test",
		EnableIstio: true,
	})

	req1 := newReconciliationRequest("oneagent1")
	req2 := newReconciliationRequest("oneagent2")

	// Operations on the CommunicationHosts list applies to both OneAgent objects, but that is fine, since that
	// is the most common use case as well, i.e., customers using multiple OneAgent objects for different
	// environments.

	// For the first reconciliation, we only create Istio objects for the API URL.
	_, err = e.Reconciler.Reconcile(req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 2, 2)

	// Once the API URL is open, we create Istio objects for each communication endpoint.
	_, err = e.Reconciler.Reconcile(req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 6, 6)

	// Add a new communication endpoint.
	e.CommunicationHosts = append(e.CommunicationHosts, "https://endpoint3.dev.ruxitlabs.com/communication")
	_, err = e.Reconciler.Reconcile(req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 8, 8)

	// Remove two communication endpoints.
	e.CommunicationHosts = e.CommunicationHosts[2:]
	_, err = e.Reconciler.Reconcile(req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 4, 4)
}

// assertIstioObjects confirms that we have the expected number of ServiceEntry and VirtualService objects, set by ese and evs respectively.
func assertIstioObjects(t *testing.T, c client.Client, ese, evs int) {
	var lst []string

	lst = findServiceEntries(t, c)
	assert.Equal(t, ese, len(lst), "unexpected number of ServiceEntry objects: %v", lst)

	lst = findVirtualServices(t, c)
	assert.Equal(t, evs, len(lst), "unexpected number of VirtualService objects: %v", lst)
}

func findServiceEntries(t *testing.T, c client.Client) []string {
	var lst istiov1alpha3.ServiceEntryList
	assert.NoError(t, c.List(context.TODO(), &client.ListOptions{}, &lst), "failed to query ServiceEntry objects")

	var out []string
	for _, x := range lst.Items {
		out = append(out, x.Spec.Hosts...)
	}
	return out
}

func findVirtualServices(t *testing.T, c client.Client) []string {
	var lst istiov1alpha3.VirtualServiceList
	assert.NoError(t, c.List(context.TODO(), &client.ListOptions{}, &lst), "failed to query VirtualService objects")

	var out []string
	for _, x := range lst.Items {
		out = append(out, x.Spec.Hosts...)
	}
	return out
}
