package v1alpha3

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
)

func Test_VirtualService(t *testing.T) {
	buffer := bytes.NewBufferString("{\"apiVersion\":\"networking.istio.io/v1alpha3\",\"kind\":\"VirtualService\",\"metadata\":{\"clusterName\":\"\",\"creationTimestamp\":\"2018-11-26T03:19:57Z\",\"generation\":1,\"name\":\"test-virtual-service\",\"namespace\":\"istio-system\",\"resourceVersion\":\"1297970\",\"selfLink\":\"/apis/networking.istio.io/v1alpha3/namespaces/istio-system/virtualservices/test-virtual-service\",\"uid\":\"266fdacc-f12a-11e8-9e1d-42010a8000ff\"},\"spec\":{\"gateways\":[\"test-gateway\"],\"hosts\":[\"*\"],\"http\":[{\"match\":[{\"uri\":{\"prefix\":\"/\"}}],\"route\":[{\"destination\":{\"host\":\"test-service\",\"port\":{\"number\":8080}}}],\"timeout\":\"10s\"}]}}\n")

	vs := VirtualService{}
	err := json.Unmarshal(buffer.Bytes(), &vs)
	assert.Equal(t, nil, err, "Could not unmarshal message")
	vss := vs.GetSpecMessage().(*istiov1alpha3.VirtualService)

	assert.Equal(t, "networking.istio.io/v1alpha3", vs.TypeMeta.APIVersion)
	assert.Equal(t, "VirtualService", vs.TypeMeta.Kind)
	assert.Equal(t, "test-virtual-service", vs.GetObjectMeta().GetName())
	assert.Equal(t, []string{"test-gateway"}, vss.GetGateways())
	assert.Equal(t, "/", vss.GetHttp()[0].GetMatch()[0].GetUri().GetPrefix())
	assert.Equal(t, "test-service", vss.GetHttp()[0].GetRoute()[0].GetDestination().GetHost())
	assert.Equal(t, &types.Duration{Seconds: 10}, vss.GetHttp()[0].GetTimeout())
}
