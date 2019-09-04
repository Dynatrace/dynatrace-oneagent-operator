package oneagent

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis"
	fakeistio "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/clientset/versioned/fake"
	istiov1alpha3 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/networking/istio/v1alpha3"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/istio"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
