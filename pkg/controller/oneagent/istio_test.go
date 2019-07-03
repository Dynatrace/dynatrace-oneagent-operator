package oneagent

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	fakeIstio "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned/fake"
	istioV1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	testAPIUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	name       = "dynatrace-oneagent"
	namespace  = "dynatrace"
)

func TestReconcileOneAgent_CreateIstioObjects(t *testing.T) {

	buffer := bytes.NewBufferString("{\"apiVersion\":\"networking.istio.io/v1alpha3\",\"kind\":\"VirtualService\",\"metadata\":{\"clusterName\":\"\",\"creationTimestamp\":\"2018-11-26T03:19:57Z\",\"generation\":1,\"name\":\"test-virtual-service\",\"namespace\":\"istio-system\",\"resourceVersion\":\"1297970\",\"selfLink\":\"/apis/networking.istio.io/v1alpha3/namespaces/istio-system/virtualservices/test-virtual-service\",\"uid\":\"266fdacc-f12a-11e8-9e1d-42010a8000ff\"},\"spec\":{\"gateways\":[\"test-gateway\"],\"hosts\":[\"*\"],\"http\":[{\"match\":[{\"uri\":{\"prefix\":\"/\"}}],\"route\":[{\"destination\":{\"host\":\"test-service\",\"port\":{\"number\":8080}}}],\"timeout\":\"10s\"}]}}\n")

	vs := istioV1alpha3.VirtualService{}
	err := json.Unmarshal(buffer.Bytes(), &vs)

	ic := fakeIstio.NewSimpleClientset(&vs)

	vsList, err := ic.NetworkingV1alpha3().VirtualServices("istio-system").List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", namespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestReconcileOneAgent_BuildDynatraceVirtualService(t *testing.T) {
	os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace)

	buffer := istio.BuildVirtualService("dt-vs", "ENVIRONMENTID.live.dynatrace.com", 443, "https")
	vs := istioV1alpha3.VirtualService{}
	err := json.Unmarshal(buffer, &vs)
	if err != nil {
		t.Errorf("Failed to marshal json %s", err)
	}
	ic := fakeIstio.NewSimpleClientset(&vs)
	vsList, err := ic.NetworkingV1alpha3().VirtualServices(namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", namespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestReconcileOneAgent_ReconcileIstioViaDynatraceClient(t *testing.T) {
	oa := newOneAgentSpec()
	oa.ApiUrl = testAPIUrl
	oa.Tokens = "token_test"
	oa.EnableIstio = true
	dynatracev1alpha1.SetDefaults_OneAgentSpec(oa)

	reconcileOA, _, server := setupReconciler(t, oa)
	defer server.Close()

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

	// rerun reconcile instio configuration update
	instance := &dynatracev1alpha1.OneAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *oa,
	}
	dtc, _ := mockBuildDynatraceClient(instance)
	commHosts, _ := dtc.GetCommunicationHosts()
	commHosts = append(commHosts, dtclient.CommunicationHost{
		Protocol: "https",
		Host:     "https://endpoint3.dev.ruxitlabs.com/communication",
		Port:     443,
	})

	var log = logf.ZapLoggerTo(os.Stdout, true)

	upd, ok := reconcileOA.reconcileIstio(log, instance, dtc)
	if !upd {
		t.Error("expected true got false, communication endpoints needed to be updated")
	}
	if !ok {
		t.Error("expected true got false, communication endpoints needed to be updated")
	}
}
