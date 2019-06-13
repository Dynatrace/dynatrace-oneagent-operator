package istio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	restclient "k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
