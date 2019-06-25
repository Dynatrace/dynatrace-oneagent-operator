package v1alpha3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
)

func Test_ServiceEntry(t *testing.T) {
	buffer := bytes.NewBufferString(`{
		"apiVersion":"networking.istio.io/v1alpha3",
		"kind":"ServiceEntry",
		"metadata":{
			"name":"test-service-entry",
			"namespace":"istio-system"
		},
		"spec":{
			"hosts":[]
		}
	}`)

	serviceEntry := ServiceEntry{}
	err := json.Unmarshal(buffer.Bytes(), &serviceEntry)
	assert.Equal(t, nil, err, "Could not unmarshal message")
	vss := serviceEntry.GetSpecMessage().(*istiov1alpha3.ServiceEntry)
	fmt.Println(vss)
	assert.Equal(t, "networking.istio.io/v1alpha3", serviceEntry.TypeMeta.APIVersion)
	assert.Equal(t, "ServiceEntry", serviceEntry.TypeMeta.Kind)
	assert.Equal(t, "test-service-entry", serviceEntry.GetObjectMeta().GetName())
}
