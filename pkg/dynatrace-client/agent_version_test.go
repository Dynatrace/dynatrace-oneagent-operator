package dynatrace_client

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	apiToken  = "some-API-token"
	paasToken = "some-PaaS-token"

	goodIp    = "192.168.0.1"
	unsetIp   = "192.168.100.1"
	unknownIp = "127.0.0.1"
)

const hostsResponse = `[
  {
	"entityId": "dynatraceSampleEntityId",
    "displayName": "good",
    "ipAddresses": [
      "10.11.12.13",
      "192.168.0.1"
    ],
    "agentVersion": {
      "major": 1,
      "minor": 142,
      "revision": 0,
      "timestamp": "20180313-173634"
    }
  },
  {
    "displayName": "unset version",
    "ipAddresses": [
      "192.168.100.1"
    ]
  }
]`

func TestAgentVersion_GetLatestAgentVersion(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	skipCert := SkipCertificateValidation(true)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	{
		_, err := dynatraceClient.GetLatestAgentVersion("", InstallerTypeDefault)

		assert.Error(t, err, "empty OS")
	}
	{
		_, err := dynatraceClient.GetLatestAgentVersion(OsUnix, "")

		assert.Error(t, err, "empty installer type")
	}
	{
		latestAgentVersion, err := dynatraceClient.GetLatestAgentVersion(OsUnix, InstallerTypeDefault)

		assert.NoError(t, err)
		assert.Equal(t, "17", latestAgentVersion, "latest agent version equals expected version")
	}
}

func TestAgentVersion_GetAgentVersionForIP(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	skipCert := SkipCertificateValidation(true)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	{
		_, err := dynatraceClient.GetAgentVersionForIP("")

		assert.Error(t, err, "lookup empty ip")
	}
	{
		_, err := dynatraceClient.GetAgentVersionForIP(unknownIp)

		assert.Error(t, err, "lookup unknown ip")
	}
	{
		_, err := dynatraceClient.GetAgentVersionForIP(unsetIp)

		assert.Error(t, err, "lookup unset ip")
	}
	{
		version, err := dynatraceClient.GetAgentVersionForIP(goodIp)

		assert.NoError(t, err, "lookup good ip")
		assert.Equal(t, "1.142.0.20180313-173634", version, "version matches for lookup good ip")
	}
}

func dynatraceServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handleRequest(r, w)
	};
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

func handleVersionForIP(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(hostsResponse))
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleLatestAgentVersion(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		out, _ := json.Marshal(map[string]string{"latestAgentVersion": "17"})
		_, _ = writer.Write(out)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}
