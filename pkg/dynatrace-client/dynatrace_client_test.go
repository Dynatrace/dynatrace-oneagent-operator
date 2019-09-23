package dynatrace_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	testAPIToken  = "testAPIToken"
	testPAASToken = "testPAASToken"
)

func TestNewDynatraceClient(t *testing.T) {
	server := initMockServer(t)
	defer server.Close()
	// installerAgent := fmt.Sprintf(
	// 	"/v1/deployment/installer/agent/%s/%s/latest/metainfo?Api-token=%s",
	// 	OsUnix,
	// 	InstallerTypeDefault,
	// 	test,
	// )

	// req, err := http.NewRequest("GET", server.URL+installerAgent, nil)
	// res, err := (&http.Client{}).Do(req)
	// st.Expect(t, err, nil)
	// st.Expect(t, res.StatusCode, 200)
	skipCert := SkipCertificateValidation(true)
	dtc, err := NewClient(server.URL, testAPIToken, testPAASToken, skipCert)
	if err != nil {
		t.Error(err)
	}

	testDynatraceClientGetLatestAgentVersion(t, dtc)
}

func initMockServer(t *testing.T) *httptest.Server {

	installerAgent := fmt.Sprintf(
		"/v1/deployment/installer/agent/%s/%s/latest/metainfo",
		OsUnix,
		InstallerTypeDefault,
	)

	testServer := httptest.NewServer(

		// NewServer takes a handler.
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			switch r.URL.Path {
			case installerAgent:
				switch r.Method {
				case "GET":
					if r.FormValue("Api-token") != "" {
						response := map[string]string{"latest": "17"}
						out, _ := json.Marshal(response)
						w.WriteHeader(http.StatusOK)
						w.Write(out)
						w.Header().Set("Content-Type", "application/json")
					}
				default:
					w.WriteHeader(http.StatusNotFound)
					w.Header().Set("Content-Type", "application/json")
				}
			default:
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "application/json")
			}
		}),
	)

	return testServer
}
