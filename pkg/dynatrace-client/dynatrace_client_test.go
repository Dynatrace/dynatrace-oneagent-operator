package dynatrace_client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDynatraceClient(t *testing.T) {

	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	skipCert := SkipCertificateValidation(true)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	testAgentVersionGetLatestAgentVersion(t, dynatraceClient)
	testAgentVersionGetAgentVersionForIP(t, dynatraceClient)
	testCommunicationHostsGetCommunicationHosts(t, dynatraceClient)
	testSendEvent(t, dynatraceClient)
}

func dynatraceServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "GET":
			if r.FormValue("Api-Token") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			handleRequest(r, w)
		case "POST":
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			handleRequest(r, w)
		}
	}
}

func handleRequest(request *http.Request, writer http.ResponseWriter) {
	latestAgentVersion := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest/metainfo", OsUnix, InstallerTypeDefault)
	versionForIP := fmt.Sprint("/v1/entity/infrastructure/hosts")
	communicationHosts := fmt.Sprint("/v1/deployment/installer/agent/connectioninfo")
	sendEvent := fmt.Sprint("/v1/events")

	switch request.URL.Path {
	case latestAgentVersion:
		handleLatestAgentVersion(request, writer)
	case versionForIP:
		handleVersionForIP(request, writer)
	case communicationHosts:
		handleCommunicationHosts(request, writer)
	case sendEvent:
		handleSendEvent(request, writer)
	default:
		writer.WriteHeader(http.StatusBadRequest)
	}
}
