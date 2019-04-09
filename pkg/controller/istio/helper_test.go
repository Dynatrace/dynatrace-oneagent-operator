package istio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	restclient "k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func initMockServer(t *testing.T) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var list interface{}
		switch req.URL.Path {
		case "/apis":
			list = &metav1.APIGroupList{
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
		default:
			t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(list)
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

	server := initMockServer(t)
	defer server.Close()

	cfg := &restclient.Config{Host: server.URL}
	r, e := CheckIstioEnabled(cfg)
	if r != true {
		t.Fail()
		t.Error(e)
	}
}
