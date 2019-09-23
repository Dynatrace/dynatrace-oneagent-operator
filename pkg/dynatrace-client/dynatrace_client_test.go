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
}

func dynatraceServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handleRequest(r, w)
	}
}

func handleRequest(request *http.Request, writer http.ResponseWriter) {
	latestAgentVersion := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest/metainfo", OsUnix, InstallerTypeDefault)
	versionForIP := fmt.Sprintf("/v1/entity/infrastructure/hosts")

	switch request.URL.Path {
	case latestAgentVersion:
		handleLatestAgentVersion(request, writer)
	case versionForIP:
		handleVersionForIP(request, writer)
	default:
		writer.WriteHeader(http.StatusBadRequest)
	}
}
