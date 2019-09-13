package istio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

func initMockServer(t *testing.T, list *metav1.APIGroupList) *httptest.Server {
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

func TestIstioEnabled(t *testing.T) {
	// resource is enabled
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
	server := initMockServer(t, list)
	defer server.Close()
	cfg := &restclient.Config{Host: server.URL}

	r, e := CheckIstioEnabled(cfg)
	if r != true {
		t.Error(e)
	}
}

func TestIstioDisabled(t *testing.T) {
	// resource is not enabled
	list := &metav1.APIGroupList{
		Groups: []metav1.APIGroup{
			{
				Name: "not.istio.group",
				Versions: []metav1.GroupVersionForDiscovery{
					{GroupVersion: "v1alpha3", Version: "v1alpha3"},
					{GroupVersion: "v1alpha3", Version: "v1alpha3"},
				},
			},
		},
	}
	server := initMockServer(t, list)
	defer server.Close()
	cfg := &restclient.Config{Host: server.URL}

	r, e := CheckIstioEnabled(cfg)
	if r != false && e == nil {
		t.Errorf("expected false, got true, %v", e)
	}
}

func TestIstioWrongConfig(t *testing.T) {
	// wrong config, we get error
	list := &metav1.APIGroupList{}
	server := initMockServer(t, list)
	defer server.Close()
	cfg := &restclient.Config{Host: "localhost:1000"}

	r, e := CheckIstioEnabled(cfg)
	if r == false && e != nil { // only true success case
		t.Logf("expected false and error %v", e)
	} else {
		t.Error("got true, expected false with error")
	}
}

func TestServiceEntryGeneration(t *testing.T) {
	// TODO: don't use environment variable on BuildServiceEntry
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")

	assert.Equal(t, `{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "ServiceEntry",
    "metadata": {
        "name": "com1",
        "namespace": "dynatrace"
    },
    "spec": {
        "hosts": [ "comtest.com" ],
        "location": "MESH_EXTERNAL",
        "ports": [{
            "name": "https-9999",
            "number": 9999,
            "protocol": "HTTPS"
        }],
        "resolution": "DNS"
    }
}`, string(BuildServiceEntry("com1", "comtest.com", 9999, "https")))

	assert.Equal(t, `{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "ServiceEntry",
    "metadata": {
        "name": "com1",
        "namespace": "dynatrace"
    },
    "spec": {
        "hosts": [ "ignored.subdomain" ],
        "addresses": [ "42.42.42.42/32" ],
        "location": "MESH_EXTERNAL",
        "ports": [{
            "name": "TCP-8888",
            "number": 8888,
            "protocol": "TCP"
        }],
        "resolution": "NONE"
    }
}`, string(BuildServiceEntry("com1", "42.42.42.42", 8888, "https")))
}

func TestVirtualServiceGeneration(t *testing.T) {
	// TODO: don't use environment variable on BuildServiceEntry
	os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace")

	assert.Equal(t, `{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "VirtualService",
    "metadata": {
        "name": "com1",
        "namespace": "dynatrace"
    },
    "spec": {
        "hosts": [ "comtest.com" ],
        "tls": [{
            "match": [{
                "port": 8888,
                "sni_hosts": [ "comtest.com" ]
            }],
            "route": [{
                "destination": {
                    "host": "comtest.com",
                    "port": { "number": 8888 }
                }
            }]
        }]
    }
}`, string(BuildVirtualService("com1", "comtest.com", 8888, "https")))

	assert.Equal(t, `{
    "apiVersion": "networking.istio.io/v1alpha3",
    "kind": "VirtualService",
    "metadata": {
        "name": "com1",
        "namespace": "dynatrace"
    },
    "spec": {
        "hosts": [ "comtest.com" ],
        "http": [{
            "match": [{
                "port": 7777
            }],
            "route": [{
                "destination": {
                    "host": "comtest.com",
                    "port": { "number": 7777 }
                }
            }]
        }]
    }
}`, string(BuildVirtualService("com1", "comtest.com", 7777, "http")))

	assert.Nil(t, BuildVirtualService("com1", "42.42.42.42", 8888, "HTTP"))
}
