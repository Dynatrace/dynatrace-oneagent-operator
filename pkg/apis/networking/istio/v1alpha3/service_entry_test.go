/*
Copyright 2018-2019 Dynatrace LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
